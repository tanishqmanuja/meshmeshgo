package meshmesh

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"os"

	gra "leguru.net/m/v2/graph"
	"leguru.net/m/v2/logger"
)

const StartAddress uint32 = 0x80000
const SectorSize uint32 = 4096
const ChunkSize uint32 = 1024

type FirmwareUploadProcedure struct {
	serial        *SerialConnection
	network       *gra.Network
	nodeid        MeshNodeId
	filename      string
	firmware      []byte
	firmwareIndex uint32
	memoryAddress uint32
	errFatal      error
	errWarn       error
	complete      bool
}

func (f *FirmwareUploadProcedure) checkMd5(md5 []byte, length uint32) (bool, bool, error) {
	reply, err := f.serial.SendReceiveApiProt(FlashGetMd5ApiRequest{Address: f.memoryAddress, Length: length}, UnicastProtocol, f.nodeid)
	if err != nil {
		return false, false, err
	}

	replyMd5 := reply.(FlashGetMd5ApiReply)
	if replyMd5.Erased {
		hexMd5 := hex.EncodeToString(replyMd5.MD5[:])
		if hexMd5 != "d41d8cd98f00b204e9800998ecf8427e" {
			logger.Log().Warn("memory erased but md5 is " + hexMd5)
		}
		return true, true, nil
	}

	equal := bytes.Equal(replyMd5.MD5, md5[:])
	return equal, replyMd5.Erased, nil
}

func (f *FirmwareUploadProcedure) Init(filename string) error {

	finfo, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return errors.New("file not found")
	}

	if finfo.IsDir() {
		return errors.New("file is a directory")
	}

	f.filename = filename
	firmware, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer firmware.Close()

	f.firmware = make([]byte, finfo.Size())
	readed, err := firmware.Read(f.firmware)
	if err != nil {
		return err
	}
	if readed != len(f.firmware) {
		return errors.New("failed to read firmware file")
	}

	f.firmwareIndex = 0
	f.memoryAddress = StartAddress
	f.errFatal = nil
	f.errWarn = nil
	f.complete = false

	return nil
}

/*
Returns:

	true if the firmware uploaded is complete
	error if a recoverable error occurs
	error if a non recoverable error occurs
*/
func (f *FirmwareUploadProcedure) Step() (bool, error, error) {
	if f.errFatal != nil || f.complete {
		return f.complete, nil, f.errFatal
	}

	if f.firmwareIndex >= uint32(len(f.firmware)) {
		f.complete = true
		return f.complete, nil, nil
	}

	sector := f.firmware[f.firmwareIndex:min(f.firmwareIndex+SectorSize, uint32(len(f.firmware)))]
	firmwareSectorMd5 := md5.Sum(sector)

	equal, erased, err := f.checkMd5(firmwareSectorMd5[:], uint32(len(sector)))
	if err != nil {
		f.errWarn = err
		return false, f.errWarn, nil
	}

	if equal {
		memoryOffset := uint32(len(sector))
		f.firmwareIndex += memoryOffset
		f.memoryAddress += memoryOffset
		return false, nil, nil
	}

	if !erased {
		reply, err := f.serial.SendReceiveApiProtTimeout(FlashEraseApiRequest{Address: f.memoryAddress, Length: SectorSize}, UnicastProtocol, f.nodeid, 1000)
		if err != nil {
			f.errWarn = errors.Join(errors.New("flash erase failed"), err)
			return false, f.errWarn, nil
		}

		replyErase := reply.(FlashEraseApiReply)
		if replyErase.Erased == 0 {
			f.errFatal = errors.New("flash erase failed")
			return false, nil, f.errFatal
		}
	}

	memoryOffset := uint32(0)
	sectorChunks := uint32((len(sector)-1)/int(ChunkSize)) + 1
	for i := uint32(0); i < sectorChunks; i++ {
		chunk := sector[i*ChunkSize : min(i*ChunkSize+ChunkSize, uint32(len(sector)))]
		reply, err := f.serial.SendReceiveApiProtTimeout(FlashWriteApiRequest{Address: f.memoryAddress + memoryOffset, Data: chunk}, UnicastProtocol, f.nodeid, 5000)
		if err != nil {
			f.errWarn = errors.Join(errors.New("flash write failed"), err)
			return false, f.errWarn, nil
		}

		replyWrite := reply.(FlashWriteApiReply)
		if replyWrite.Result {
			f.errFatal = errors.New("flash write failed")
			return false, nil, f.errFatal
		}

		memoryOffset += uint32(len(chunk))
	}

	equal, _, err = f.checkMd5(firmwareSectorMd5[:], uint32(len(sector)))
	if err != nil {
		f.errWarn = err
		return false, f.errWarn, nil
	}

	if !equal {
		f.errFatal = errors.New("flash md5 sum mismatch")
		return false, nil, f.errFatal
	}

	f.firmwareIndex += memoryOffset
	f.memoryAddress += memoryOffset
	return false, nil, nil
}

func (f *FirmwareUploadProcedure) BytesSent() uint32 {
	return f.firmwareIndex
}

func (f *FirmwareUploadProcedure) BytesTotal() uint32 {
	return uint32(len(f.firmware))
}

func (f *FirmwareUploadProcedure) Percent() float64 {
	if f.firmware == nil {
		return 0
	}
	return float64(f.firmwareIndex) / float64(len(f.firmware))
}

func NewFirmwareUploadProcedure(serial *SerialConnection, network *gra.Network, nodeid MeshNodeId) *FirmwareUploadProcedure {
	return &FirmwareUploadProcedure{serial: serial, network: network, nodeid: nodeid}
}
