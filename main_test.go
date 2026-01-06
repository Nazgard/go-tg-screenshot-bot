package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Моки для интерфейсов

type mockScreenshotCapturer struct {
	captureCalled bool
	display       int
	returnFile    string
	returnReader  io.Reader
	returnErr     error
}

func (m *mockScreenshotCapturer) CaptureDisplay(display int) (string, io.Reader, error) {
	m.captureCalled = true
	m.display = display
	return m.returnFile, m.returnReader, m.returnErr
}

type mockMessageSender struct {
	sendCalled bool
	chatID     int64
	filename   string
	photo      io.Reader
	returnErr  error
}

func (m *mockMessageSender) SendPhoto(chatID int64, filename string, photo io.Reader) error {
	m.sendCalled = true
	m.chatID = chatID
	m.filename = filename
	m.photo = photo
	return m.returnErr
}

type mockLogger struct {
	printfCalled  bool
	printlnCalled bool
	fatalfCalled  bool
	lastPrintf    string
	lastPrintln   string
	lastFatalf    string
}

func (m *mockLogger) Printf(format string, v ...interface{}) {
	m.printfCalled = true
	m.lastPrintf = fmt.Sprintf(format, v...)
}

func (m *mockLogger) Println(v ...interface{}) {
	m.printlnCalled = true
	if len(v) > 0 {
		m.lastPrintln = fmt.Sprint(v[0])
	}
}

func (m *mockLogger) Fatalf(format string, v ...interface{}) {
	m.fatalfCalled = true
	m.lastFatalf = fmt.Sprintf(format, v...)
}

// Тесты

func TestIsAllowedChatID(t *testing.T) {
	tests := []struct {
		name     string
		allowed  int64
		chatID   int64
		expected bool
	}{
		{"Allowed", 12345, 12345, true},
		{"Not allowed", 12345, 99999, false},
		{"Zero allowed", 0, 0, true},
		{"Negative", -1, -1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := &Application{
				Config: &Config{AllowedChatID: tt.allowed},
			}
			assert.Equal(t, tt.expected, app.IsAllowedChatID(tt.chatID))
		})
	}
}

func TestHandleTelegramMessage_WhoamiCommand(t *testing.T) {
	mockSender := &mockMessageSender{}
	app := &Application{
		Config:        &Config{AllowedChatID: 12345},
		MessageSender: mockSender,
		Logger:        &mockLogger{},
	}

	err := app.HandleTelegramMessage("/whoami", 12345)

	require.NoError(t, err)
	assert.True(t, mockSender.sendCalled, "SendPhoto should be called for /whoami")
	assert.Equal(t, int64(12345), mockSender.chatID)
	assert.Equal(t, "whoami.png", mockSender.filename)

	// Проверяем, что в фото отправлен текст с chatID
	data, _ := io.ReadAll(mockSender.photo.(io.Reader))
	assert.Contains(t, string(data), "Your chat ID: 12345")
}

func TestHandleTelegramMessage_Unauthorized(t *testing.T) {
	mockSender := &mockMessageSender{}
	mockLog := &mockLogger{}
	app := &Application{
		Config:        &Config{AllowedChatID: 99999},
		MessageSender: mockSender,
		Logger:        mockLog,
	}

	err := app.HandleTelegramMessage("0", 12345)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unauthorized chat ID")
	assert.False(t, mockSender.sendCalled, "SendPhoto should NOT be called")
	assert.True(t, mockLog.printfCalled)
	assert.Contains(t, mockLog.lastPrintf, "Unauthorized access attempt from chat ID 12345")
}

func TestHandleTelegramMessage_InvalidDisplayNumber(t *testing.T) {
	mockSender := &mockMessageSender{}
	app := &Application{
		Config:        &Config{AllowedChatID: 12345},
		MessageSender: mockSender,
	}

	err := app.HandleTelegramMessage("not_a_number", 12345)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid display number")
	assert.False(t, mockSender.sendCalled)
}

func TestHandleTelegramMessage_SuccessfulScreenshot(t *testing.T) {
	expectedFile := "0_1920x1080.png"
	expectedData := []byte("fake png data")

	mockCapturer := &mockScreenshotCapturer{
		returnFile:   expectedFile,
		returnReader: bytes.NewBuffer(expectedData),
		returnErr:    nil,
	}
	mockSender := &mockMessageSender{}
	app := &Application{
		Config:             &Config{AllowedChatID: 12345},
		ScreenshotCapturer: mockCapturer,
		MessageSender:      mockSender,
		Logger:             &mockLogger{},
		LastDisplay:        999, // должно обновиться
	}

	err := app.HandleTelegramMessage("2", 12345)

	require.NoError(t, err)
	assert.True(t, mockCapturer.captureCalled)
	assert.Equal(t, 2, mockCapturer.display)
	assert.True(t, mockSender.sendCalled)
	assert.Equal(t, int64(12345), mockSender.chatID)
	assert.Equal(t, expectedFile, mockSender.filename)

	// Проверяем, что LastDisplay обновился
	assert.Equal(t, 2, app.LastDisplay)

	// Проверяем, что данные переданы корректно
	sentData, _ := io.ReadAll(mockSender.photo.(io.Reader))
	assert.Equal(t, expectedData, sentData)
}

func TestHandleTelegramMessage_CaptureError(t *testing.T) {
	mockCapturer := &mockScreenshotCapturer{
		returnErr: errors.New("display not found"),
	}
	mockSender := &mockMessageSender{}
	mockLog := &mockLogger{}
	app := &Application{
		Config:             &Config{AllowedChatID: 12345},
		ScreenshotCapturer: mockCapturer,
		MessageSender:      mockSender,
		Logger:             mockLog,
	}

	err := app.HandleTelegramMessage("0", 12345)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to capture screenshot")
	assert.True(t, mockCapturer.captureCalled)
	assert.False(t, mockSender.sendCalled)
}

func TestHandleTelegramMessage_SendError(t *testing.T) {
	mockCapturer := &mockScreenshotCapturer{
		returnFile:   "test.png",
		returnReader: bytes.NewBuffer([]byte("data")),
	}
	mockSender := &mockMessageSender{
		returnErr: errors.New("telegram api error"),
	}
	app := &Application{
		Config:             &Config{AllowedChatID: 12345},
		ScreenshotCapturer: mockCapturer,
		MessageSender:      mockSender,
		Logger:             &mockLogger{},
	}

	err := app.HandleTelegramMessage("0", 12345)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "telegram api error")
	assert.True(t, mockCapturer.captureCalled)
	assert.True(t, mockSender.sendCalled)
}
