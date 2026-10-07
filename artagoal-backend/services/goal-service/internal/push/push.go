// Package push mengirim notifikasi Web Push (VAPID) ke browser pelanggan.
package push

import (
	"encoding/json"
	"io"

	webpush "github.com/SherClockHolmes/webpush-go"
)

// Subscription adalah kredensial push milik satu browser.
type Subscription struct {
	Endpoint string
	P256dh   string
	Auth     string
}

// Payload adalah isi notifikasi yang diterima service worker.
type Payload struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	URL   string `json:"url"`
}

// Sender mengirim push memakai kunci VAPID aplikasi.
type Sender struct {
	publicKey  string
	privateKey string
	contact    string
}

// NewSender membuat sender; contact mis. "mailto:admin@artagoal.local".
func NewSender(publicKey, privateKey, contact string) *Sender {
	return &Sender{publicKey: publicKey, privateKey: privateKey, contact: contact}
}

// Enabled melaporkan apakah kunci VAPID terkonfigurasi.
func (s *Sender) Enabled() bool {
	return s != nil && s.publicKey != "" && s.privateKey != ""
}

// PublicKey mengembalikan kunci publik VAPID untuk klien.
func (s *Sender) PublicKey() string {
	if s == nil {
		return ""
	}
	return s.publicKey
}

// Send mengirim satu push. gone=true berarti endpoint sudah mati
// (404/410) dan langganannya sebaiknya dihapus.
func (s *Sender) Send(sub Subscription, title, body, url string) (gone bool, err error) {
	payload, err := json.Marshal(Payload{Title: title, Body: body, URL: url})
	if err != nil {
		return false, err
	}

	resp, err := webpush.SendNotification(payload, &webpush.Subscription{
		Endpoint: sub.Endpoint,
		Keys:     webpush.Keys{P256dh: sub.P256dh, Auth: sub.Auth},
	}, &webpush.Options{
		Subscriber:      s.contact,
		VAPIDPublicKey:  s.publicKey,
		VAPIDPrivateKey: s.privateKey,
		TTL:             24 * 3600,
	})
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode == 404 || resp.StatusCode == 410 {
		return true, nil
	}
	return false, nil
}
