package meshmesh

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
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

func (f *FirmwareUploadProcedure) checkMemoryMd5(md5 [16]byte, memoryAddress uint32, length uint32) (bool, bool, error) {
	reply, err := f.serial.SendReceiveApiProt(FlashGetMd5ApiRequest{Address: memoryAddress, Length: length}, UnicastProtocol, f.nodeid, f.network)
	if err != nil {
		return false, false, err
	}

	replyMd5 := reply.(FlashGetMd5ApiReply)
	if replyMd5.Erased {
		hexMd5 := hex.EncodeToString(replyMd5.MD5[:])
		if hexMd5 != "6ae59e64850377ee5470c854761551ea" {
			logger.Log().Warn("memory erased but md5 is " + hexMd5)
		}
		return false, true, nil
	}

	equal := bytes.Equal(replyMd5.MD5, md5[:])
	return equal, replyMd5.Erased, nil
}

func (f *FirmwareUploadProcedure) checkMd5(data []byte, memoryAddress uint32, length uint32) (bool, bool, error) {
	md5 := md5.Sum(data)
	equal, erased, err := f.checkMemoryMd5(md5, memoryAddress, length)
	if err != nil {
		return false, false, err
	}
	if !equal {
		return false, false, errors.New("md5 sum mismatch")
	}
	return equal, erased, nil
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
		// Upload complete, check md5 sum of the whole firmware
		equal, _, err := f.checkMd5(f.firmware, StartAddress, uint32(len(f.firmware)))
		if err != nil || !equal {
			f.errFatal = errors.Join(errors.New("error computing md5 sum of memory"), err)
			return equal, nil, f.errFatal
		}

		_, err = f.serial.SendReceiveApiProt(FlashEBootApiRequest{Address: StartAddress, Length: uint32(len(f.firmware))}, UnicastProtocol, f.nodeid, f.network)
		if err != nil {
			f.errFatal = errors.Join(errors.New("flash eboot failed"), err)
			return false, nil, f.errFatal
		}
		f.complete = true
		return f.complete, nil, nil
	}

	sector := f.firmware[f.firmwareIndex:min(f.firmwareIndex+SectorSize, uint32(len(f.firmware)))]
	sectorOffset := uint32(len(sector))

	firmwareSectorMd5 := md5.Sum(sector)
	equal, erased, err := f.checkMemoryMd5(firmwareSectorMd5, f.memoryAddress, sectorOffset)
	if err != nil {
		f.errWarn = err
		return false, f.errWarn, nil
	}

	logger.WithFields(logger.Fields{
		"firmwareIndex": fmt.Sprintf("%08X", f.firmwareIndex),
		"memoryAddress": fmt.Sprintf("%08X", f.memoryAddress-StartAddress),
		"sectorSize":    fmt.Sprintf("%d", sectorOffset),
		"equal":         equal,
		"erased":        erased,
	}).Info("firmware sector already uploaded")

	if equal {
		f.firmwareIndex += sectorOffset
		f.memoryAddress += sectorOffset
		return false, nil, nil
	}

	if !erased {
		reply, err := f.serial.SendReceiveApiProtTimeout(FlashEraseApiRequest{Address: f.memoryAddress, Length: SectorSize}, UnicastProtocol, f.nodeid, f.network, 1000)
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

	sectorOffset = uint32(0)
	sectorChunks := uint32((len(sector)-1)/int(ChunkSize)) + 1
	for i := uint32(0); i < sectorChunks; i++ {
		chunk := sector[i*ChunkSize : min(i*ChunkSize+ChunkSize, uint32(len(sector)))]
		reply, err := f.serial.SendReceiveApiProtTimeout(FlashWriteApiRequest{Address: f.memoryAddress + sectorOffset, Data: chunk}, UnicastProtocol, f.nodeid, f.network, 5000)
		if err != nil {
			f.errWarn = errors.Join(errors.New("flash write failed"), err)
			return false, f.errWarn, nil
		}

		replyWrite := reply.(FlashWriteApiReply)
		if replyWrite.Result {
			f.errFatal = errors.New("flash write failed")
			return false, nil, f.errFatal
		}

		sectorOffset += uint32(len(chunk))
	}

	equal, _, err = f.checkMemoryMd5(firmwareSectorMd5, f.memoryAddress, sectorOffset)
	if err != nil {
		f.errWarn = err
		return false, f.errWarn, nil
	}

	if !equal {
		f.errFatal = errors.New("flash md5 sum mismatch")
		return false, nil, f.errFatal
	}

	f.firmwareIndex += sectorOffset
	f.memoryAddress += sectorOffset
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

func (f *FirmwareUploadProcedure) Run() error {
	err := f.Init(f.filename)
	if err != nil {
		return err
	}

	for {
		complete, err, errFatal := f.Step()
		if errFatal != nil {
			return errFatal
		}
		if complete {
			return err
		}
	}
}

func NewFirmwareUploadProcedure(serial *SerialConnection, network *gra.Network, nodeid MeshNodeId) *FirmwareUploadProcedure {
	return &FirmwareUploadProcedure{serial: serial, network: network, nodeid: nodeid}
}
