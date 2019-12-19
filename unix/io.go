package unix

import (
	"net"
	"os"
	"fmt"

	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

type Server struct {}

func (s *Server) Init(addr string, handler func(*net.UnixConn) error) error {

	if err := os.RemoveAll(addr); err != nil {
		logrus.Errorf("Unable to remove address %v (%v).", addr, err)
		return err
    }

	unixAddr, err := net.ResolveUnixAddr("unix", addr)
	if err != nil {
		logrus.Errorf("Unable to resolve address %v (%v).", addr, err)
		return err
	}

	listener, err := net.ListenUnix("unix", unixAddr)
	if err != nil {
		logrus.Errorf("Unable to listen through addr %v (%v).", addr, err)
		return err
	}
	defer listener.Close()

	err = os.Chmod(addr, 0700)
	if err != nil {
		logrus.Errorf("Unable to set %v socket permissions (%v).", addr, err)
		return err
    }

	for {
		//
		conn, err := listener.AcceptUnix()
		if err != nil {
			logrus.Errorf("Unable to establish connection (%v).", err)
			logrus.Errorf("hey")
			return err
		}

		go handler(conn)
	}

	return nil
}

type Client struct {}

func (c *Client) Connect(addr string) (*net.UnixConn, error) {

	unixAddr, err := net.ResolveUnixAddr("unix", addr)
	if err != nil {
		logrus.Errorf("Unable to resolve address %v (%v).", addr, err)
		return nil, err
	}

	conn, err := net.DialUnix("unix", nil, unixAddr)
	if err != nil {
		logrus.Errorf("Unable to dial to addr %v (%v).", addr, err)
		return nil, err
	}
	defer conn.Close()

	return conn, nil
}

const inbLength = 256
var oobLength = unix.CmsgSpace(4)

func RecvGenericMsg(c *net.UnixConn, inb []byte, oob []byte) error {

	inbSize := len(inb)
	oobSize := len(oob)

	inbn, oobn, _, _, err := c.ReadMsgUnix(inb, oob)
	if err != nil {
		logrus.Errorf("Unable to read message from endpoint %v", c.RemoteAddr())
		return err
	}

	if inbn >= inbSize || oobn >= oobSize {
		logrus.Errorf("Invalid msg received from endpoint %v", c.RemoteAddr())
		return err
	}

	// Truncate inband and outbound buffers to match received sizes.
	inb = inb[:inbn]
	oob = oob[:oobn]
	
	return nil
}

func SendGenericMsg(c *net.UnixConn, inb []byte, oob []byte) error {

	inbSize := len(inb)
	oobSize := len(oob)

	inbn, oobn, err := c.WriteMsgUnix(inb, oob, c.RemoteAddr().(*net.UnixAddr))
	if err != nil {
		logrus.Errorf("Unable to write message to endpoint %v", c.RemoteAddr())
		return err
	}

	if inbn < inbSize || oobn < oobSize {
		logrus.Errorf("Invalid msg sent to endpoint %v", c.RemoteAddr())
		return err
	}

	return nil
}

func parseScmRightsFd(c *net.UnixConn, oob []byte) (int, error) {

    scms, err := unix.ParseSocketControlMessage(oob)
    if err != nil {
        logrus.Errorf("Unexpected error while parsing SocketControlMessage msg")
        return 0, err
    }
	if len(scms) != 1 {
        logrus.Errorf("Unexpected number of SocketControlMessages received: expected 1, received %v",
			len(scms))
        return 0, err
	}

	fds, err := unix.ParseUnixRights(&scms[0])
	if err != nil {
		return 0, err
	}
	if len(fds) != 1 {
		return -1, fmt.Errorf("Unexpected number of file-descriptors received: expected 1, received %v",
			len(fds))
	}
	fd := int(fds[0])

	return fd, nil
}

func RecvSeccompNotifMsg(c *net.UnixConn) (int, string, error) {

	inb := make([]byte, inbLength)
	oob := make([]byte, oobLength)

	if err := RecvGenericMsg(c, inb, oob); err != nil {
		return -1, "", err
	}

	// Parse received control-msg to extract one file-descriptor.
	fd, err := parseScmRightsFd(c, oob)
	if err != nil {
		return -1, "", err
	}

	payload := string(oob)

	return fd, payload, nil
}

func SendSeccompNotifMsg(c *net.UnixConn, fd int, cntrId string) error {

	err := SendGenericMsg(c, []byte(string(fd)), []byte(cntrId))
	if err != nil {
		return err
	}

	return nil
}

func (s *Server) SendMsg(socket int, fds []int) error {

    rights := unix.UnixRights(fds...)
    err := unix.Sendmsg(socket, nil, rights, nil, 0)
    if err != nil {
        logrus.Errorf("Error while sending SocketControlMessage")
        return err
    }

    return nil
}

//     buf := make([]byte, syscall.CmsgSpace(fdsNum * strconv.IntSize))
    
//     _, _, _, _, err := syscall.Recvmsg(socket, nil, buf, 0)
//     if err != nil {
//         return nil, err
//     }

//     scms, err := syscall.ParseSocketControlMessage(buf)
//     if err != nil {
//         logrus.Errorf("Unexpected error while parsing SocketControlMessage msg")
//         return nil, err
//     }
// 	if len(scms) != fdsNum {
//         logrus.Errorf("Unexpected number of SocketControlMessages received: expected %v received %v",
//         fdsNum, len(scms))
//         return nil, err
//     }

//     var rcvdFds []int
//     for _, scm := range scms {
//         fds, err := syscall.ParseUnixRights(&scm)
//         if err != nil {
//             logrus.Errorf("Unexpected error while parsing scm msg")
//             return nil, err
//         }

//         for _, fd := range fds {
//             rcvdFds = append(rcvdFds, fd)
//         }
//     }

//     return rcvdFds, nil
// }
