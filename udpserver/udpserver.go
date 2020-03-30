package udpserver

import (
	"chimney-go/privacy"
	"chimney-go/utils"
	"log"
	"net"
)

//request
// I text
//-----------------------------------------------------------------------------------------
//  4   | 1byte | I |     data zone( << 4096)
//-----------------------------------------------------------------------------------------

// raw text
//---------------------------------------------------------------------------------------
// | 1 cmd| 2(len) | 1 type|  ip(domain) target | 2(len) 1 type| ip(domain) src| (3072)data|
//
//
//-----------------------------------------------------------------------------------------

// response
//-----------------------------------------------------------------------------------------
// 1 answer |2(len) | 1 type|  ip(domain) target | 2(len) 1 type| ip(domain) src| data(3072)

type udpproxy struct {
	listen string
	I      privacy.EncryptThings
	key    []byte
}

// UDPServer ..
type UDPServer interface {
	Run()
}

// NewUDPServer ...
func NewUDPServer(listenAddress string, i privacy.EncryptThings) UDPServer {
	return &udpproxy{
		listen: listenAddress,
		I:      i,
	}
}

func (s *udpproxy) Run() {

	udpaddr, err := net.ResolveUDPAddr("udp", s.listen)
	if err != nil {
		log.Println(" UDP addres resolved failed", err)
		return
	}

	socket, err := net.ListenUDP("udp", udpaddr)
	if err != nil {
		log.Println(" can not listen on ", s.listen, " to recv client request.")
		return
	}
	go func(udp *net.UDPConn) {
		defer func() {
			if err := recover(); err != nil {
				log.Println(" fatal error on udp server: ", err)
			}
		}()
		defer udp.Close()

		for {
			data := WantAPiece()
			n, rAddr, err := udp.ReadFromUDP(data)
			if err != nil {
				log.Println(" read info failed ", err)
				continue
			}

			go s.serveOne(data, rAddr, n, udp)
		}

	}(socket)
}

func (s *udpproxy) serveOne(buf []byte, addr *net.UDPAddr, n int, udp *net.UDPConn) {
	defer func() {
		ReturnAPice(buf)
	}()

	data := buf[:n]

	dataL := utils.Bytes2Int(data[:4])
	if dataL != uint32(len(data[4:])) {
		log.Println("partition of data was lost!!! ")
		return
	}

	I, err := privacy.FromBytes(data[5 : 5+data[4]])
	if err != nil {
		log.Println("got everything  failed", err)
		return
	}

	out, err := I.Uncompress(data[5+data[4]:], s.key)
	if err != nil {
		log.Println("uncompressed failed", err)
		return
	}
	sendata, err := ParseData(out)
	if err != nil {
		log.Println("uncompressed failed", err)
		return
	}

	socket, err := net.Dial("udp", sendata.dst.String())
	if err != nil {
		log.Println("dial udp failed  ", sendata.dst)
		return
	}

	defer func() {
		socket.Close()
	}()

	_, err = socket.Write(sendata.data)
	if err != nil {
		log.Println("dial udp failed  ", sendata.dst)
		return
	}

	if sendata.cmd == 0 {
		log.Println("need not response ", sendata.dst)
		return
	}

	readBuffer := WantAPiece()
	defer func() {
		ReturnAPice(readBuffer)
	}()

	n, err = socket.Read(readBuffer[:len(readBuffer)-512])
	if err != nil {
		log.Println("read from target failed ", err)
		return
	}

	answser := &UDPCom{
		src:  sendata.dst,
		dst:  sendata.src,
		cmd:  1,
		data: readBuffer[:n],
	}

	ll := ToAnswer(answser)

	out, err = s.I.Compress(ll, s.key)
	if err != nil {
		log.Println("Compress failed ", err)
		return
	}

	pkg := WantAPiece()
	defer func() {
		ReturnAPice(pkg)
	}()

	ibuf := s.I.ToBytes()
	lens := len(out) + len(ibuf) + 1

	if lens > len(pkg)-4 {
		log.Println("out data too big ", "HAHAH")
		return
	}

	lensf := utils.Int2Bytes(uint32(lens))

	copy(pkg, lensf)
	pkg[4] = byte(len(ibuf))
	copy(pkg[5:], ibuf)
	copy(pkg[5+len(ibuf):], out)

	_, err = udp.WriteToUDP(pkg[:lens+4], addr)

	log.Println("write to result", err)
}
