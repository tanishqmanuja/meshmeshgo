package meshmesh

import "errors"

func UploadFirmware(target MeshNodeId, firmware string, serial *SerialConnection) error {
	reply1, err := serial.SendReceiveApiProt(EchoApiRequest{Echo: "CIAO"}, UnicastProtocol, target)
	if err != nil {
		return err
	}
	echo, ok := reply1.(EchoApiReply)
	if !ok {
		return errors.New("invalid echo reply type")
	}
	if echo.Echo != "CIAO" {
		return errors.New("invalid echo reply")
	}

	return nil
}
