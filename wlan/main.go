package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"
)

const (
	magicPacketSize         = 6
	macRepetitions          = 16
	macLen                  = 12
	defaultAttempts         = 3
	macAddressLength        = 6
	defaultBroadcastAddress = "255.255.255.255:9"
	networkProtocol         = "udp"
)

// MAC-адрес
type MACAddress [macAddressLength]byte

// парсинг строки MAC-адреса
func ParseMAC(s string) (MACAddress, error) {
	var mac MACAddress

	// удаление разделителей
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, ":", "")
	s = strings.ReplaceAll(s, "-", "")
	s = strings.ReplaceAll(s, ".", "")
	s = strings.ReplaceAll(s, " ", "")

	if len(s) != macLen {
		return mac, fmt.Errorf("неверная длина MAC-адреса: ожидается %d байт, получено %d", macLen, len(s))
	}

	// проверка на HEX
	bytes, err := hex.DecodeString(s)
	if err != nil {
		return mac, err
	}

	copy(mac[:], bytes)
	return mac, nil
}

// создание пакета Wake-on-LAN
func CreatePacket(mac MACAddress) []byte {
	packet := make([]byte, magicPacketSize+macRepetitions*len(mac))

	// magicPacketSize байт FF в начале
	for i := 0; i < magicPacketSize; i++ {
		packet[i] = 0xFF
	}

	// macRepetitions повторов MAC-адреса
	for i := 1; i <= macRepetitions; i++ {
		offset := magicPacketSize + i*len(mac)
		copy(packet[offset:offset+len(mac)], mac[:])
	}

	return packet
}

// отправка WLAN пакет
func SendWLANPacket(mac MACAddress, broadcastAddr string) error {
	// broadcast адрес
	udpAddr, err := net.ResolveUDPAddr(networkProtocol, broadcastAddr)
	if err != nil {
		return fmt.Errorf("ошибка разрешения адреса: %v", err)
	}

	// создание UDP соединения
	con, err := net.DialUDP(networkProtocol, nil, udpAddr)
	if err != nil {
		return fmt.Errorf("ошибка создания %s соединения: %v", networkProtocol, err)
	}
	defer con.Close()

	// создание пакета
	packet := CreatePacket(mac)

	// отправка пакета
	_, err = con.Write(packet)
	if err != nil {
		return fmt.Errorf("ошибка отправки пакета: %v", err)
	}

	return nil
}

// попытка отправить пакет несколько раз
func SendMultiple(mac MACAddress, broadcastAddr string, attempts int, delay time.Duration) error {
	for i := 0; i < attempts; i++ {
		fmt.Printf("Попытка %d из %d...\n", i+1, attempts)

		err := SendWLANPacket(mac, broadcastAddr)
		if err != nil {
			return err
		}

		if i < attempts-1 {
			time.Sleep(delay)
		}
	}
	return nil
}

func main() {
	// парсинг аргументов командной строки
	macStr := flag.String("mac", "", "MAC-адрес целевого компьютера (формат: XX:XX:XX:XX:XX:XX)")
	broadcast := flag.String("broadcast", defaultBroadcastAddress, "Broadcast адрес и порт")
	attempts := flag.Int("attempts", defaultAttempts, "Количество попыток отправки")
	flag.Parse()

	// проверка обязательных параметров
	if *macStr == "" {
		fmt.Println("Ошибка: необходимо указать MAC-адрес")
		fmt.Println("Использование: wlan -mac=XX:XX:XX:XX:XX:XX")
		fmt.Println("Пример: wlan -mac=00:1A:2B:3C:4D:5E")
		flag.PrintDefaults()
		os.Exit(1)
	}

	// парсинг MAC-адреса
	mac, err := ParseMAC(*macStr)
	if err != nil {
		log.Fatalf("Ошибка парсинга MAC-адреса: %v", err)
	}

	fmt.Printf("Отправка Wake-on-LAN пакета:\n")
	fmt.Printf("MAC-адрес: %s\n", *macStr)
	fmt.Printf("Broadcast: %s\n", *broadcast)
	fmt.Printf("Попыток: %d\n", *attempts)

	// отправка пакета несколько раз
	err = SendMultiple(mac, *broadcast, *attempts, 1*time.Second)
	if err != nil {
		log.Fatalf("Ошибка отправки пакета: %v", err)
	}

	fmt.Println("Пакет успешно отправлен")
}
