package ui

import (
	"fmt"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"p2p-chat/internal/peer"
	"p2p-chat/internal/protocol"
)

type App struct {
	fyneApp    fyne.App
	mainWindow fyne.Window
	peer       *peer.Peer

	stunURL  string
	useTLS   bool
	localIP  string

	chatHistory  *widget.List
	messages     []chatMessage
	messageInput *widget.Entry
	peerList     *widget.List
	peers        []protocol.PeerInfo
	statusLabel  *widget.Label

	currentView string
}

type chatMessage struct {
	from    string
	content string
	isFile  bool
	time    time.Time
}

func New(stunURL, localIP string, useTLS bool) *App {
	return &App{
		fyneApp:  app.New(),
		stunURL:  stunURL,
		useTLS:   useTLS,
		localIP:  localIP,
		messages: []chatMessage{},
		peers:    []protocol.PeerInfo{},
	}
}

func (a *App) Run() {
	a.mainWindow = a.fyneApp.NewWindow("P2P Chat")
	a.mainWindow.Resize(fyne.NewSize(800, 600))

	a.showLoginView()

	a.mainWindow.ShowAndRun()
}

func (a *App) showLoginView() {
	usernameEntry := widget.NewEntry()
	usernameEntry.SetPlaceHolder("Enter username")

	portEntry := widget.NewEntry()
	portEntry.SetPlaceHolder("Port (leave empty for random)")

	statusLabel := widget.NewLabel("")

	loginBtn := widget.NewButton("Connect", func() {
		username := strings.TrimSpace(usernameEntry.Text)
		if username == "" {
			statusLabel.SetText("Username is required")
			return
		}

		port := 0
		if portEntry.Text != "" {
			fmt.Sscanf(portEntry.Text, "%d", &port)
		}

		p, err := peer.New(username, a.stunURL, port, a.useTLS)
		if err != nil {
			statusLabel.SetText(fmt.Sprintf("Error: %v", err))
			return
		}

		a.peer = p
		a.setupPeerCallbacks()

		if err := a.peer.Start(); err != nil {
			statusLabel.SetText(fmt.Sprintf("Error starting peer: %v", err))
			return
		}

		if err := a.peer.Register(a.localIP); err != nil {
			statusLabel.SetText(fmt.Sprintf("Error registering: %v", err))
			return
		}

		a.showMainView()
	})

	content := container.NewVBox(
		layout.NewSpacer(),
		container.NewCenter(widget.NewLabel("P2P Chat")),
		container.NewCenter(container.NewVBox(
			widget.NewLabel("Username:"),
			usernameEntry,
			widget.NewLabel("Port (optional):"),
			portEntry,
			loginBtn,
			statusLabel,
		)),
		layout.NewSpacer(),
	)

	a.mainWindow.SetContent(content)
}

func (a *App) setupPeerCallbacks() {
	a.peer.SetCallbacks(
		func(from, content string) {
			a.addMessage(from, content, false)
		},
		func(from, filename string, size int64) {
			a.addMessage(from, fmt.Sprintf("📎 Receiving file: %s (%d bytes)", filename, size), true)
		},
		func(from, filename string) {
			a.addMessage("System", fmt.Sprintf("✅ File received: %s (saved to downloads/)", filename), true)
		},
		func(peerName string) {
			a.addMessage("System", fmt.Sprintf("%s disconnected", peerName), false)
			a.showMainView()
		},
		func(from string) bool {
			accepted := make(chan bool)
			a.mainWindow.Canvas().Refresh(a.mainWindow.Content())

			dialog.ShowConfirm(
				"Connection Request",
				fmt.Sprintf("%s wants to connect. Accept?", from),
				func(accept bool) {
					accepted <- accept
				},
				a.mainWindow,
			)

			result := <-accepted
			if result {
				a.showChatView()
			}
			return result
		},
	)
}

func (a *App) showMainView() {
	a.currentView = "main"
	a.statusLabel = widget.NewLabel(fmt.Sprintf("Connected as: %s (Port: %d)", a.peer.Username(), a.peer.Port()))

	a.peerList = widget.NewList(
		func() int { return len(a.peers) },
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewLabel(""),
				layout.NewSpacer(),
				widget.NewButton("Connect", nil),
			)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			c := obj.(*fyne.Container)
			label := c.Objects[0].(*widget.Label)
			btn := c.Objects[2].(*widget.Button)

			p := a.peers[id]
			label.SetText(fmt.Sprintf("%s (%s:%d)", p.Username, p.IP, p.Port))

			btn.OnTapped = func() {
				a.connectToPeer(p)
			}
		},
	)

	refreshBtn := widget.NewButtonWithIcon("Refresh", theme.ViewRefreshIcon(), func() {
		a.refreshPeers()
	})

	disconnectBtn := widget.NewButton("Logout", func() {
		a.peer.Close()
		a.showLoginView()
	})

	topBar := container.NewBorder(nil, nil, a.statusLabel, container.NewHBox(refreshBtn, disconnectBtn))

	content := container.NewBorder(
		container.NewVBox(topBar, widget.NewSeparator()),
		nil, nil, nil,
		container.NewVScroll(a.peerList),
	)

	a.mainWindow.SetContent(content)
	a.refreshPeers()
}

func (a *App) refreshPeers() {
	peers, err := a.peer.GetPeers()
	if err != nil {
		dialog.ShowError(err, a.mainWindow)
		return
	}

	a.peers = []protocol.PeerInfo{}
	for _, p := range peers {
		if p.Username != a.peer.Username() {
			a.peers = append(a.peers, p)
		}
	}

	a.peerList.Refresh()
}

func (a *App) connectToPeer(peerInfo protocol.PeerInfo) {
	if err := a.peer.Connect(peerInfo); err != nil {
		dialog.ShowError(err, a.mainWindow)
		return
	}
	a.showChatView()
}

func (a *App) showChatView() {
	a.currentView = "chat"
	a.messages = []chatMessage{}

	a.chatHistory = widget.NewList(
		func() int { return len(a.messages) },
		func() fyne.CanvasObject {
			return widget.NewLabel("")
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			label := obj.(*widget.Label)
			msg := a.messages[id]
			timeStr := msg.time.Format("15:04")
			label.SetText(fmt.Sprintf("[%s] %s: %s", timeStr, msg.from, msg.content))
		},
	)

	a.messageInput = widget.NewMultiLineEntry()
	a.messageInput.SetPlaceHolder("Type a message...")
	a.messageInput.Wrapping = fyne.TextWrapWord

	sendBtn := widget.NewButtonWithIcon("Send", theme.MailSendIcon(), func() {
		a.sendMessage()
	})

	fileBtn := widget.NewButtonWithIcon("File", theme.FolderOpenIcon(), func() {
		a.sendFile()
	})

	disconnectBtn := widget.NewButton("Disconnect", func() {
		a.peer.Disconnect()
		a.showMainView()
	})

	topBar := container.NewBorder(
		nil, nil,
		widget.NewLabel(fmt.Sprintf("Chatting with: %s", a.peer.ConnectedTo())),
		disconnectBtn,
	)

	inputArea := container.NewBorder(nil, nil, nil, container.NewHBox(fileBtn, sendBtn), a.messageInput)

	content := container.NewBorder(
		container.NewVBox(topBar, widget.NewSeparator()),
		container.NewVBox(widget.NewSeparator(), inputArea),
		nil, nil,
		a.chatHistory,
	)

	a.mainWindow.SetContent(content)
}

func (a *App) sendMessage() {
	content := strings.TrimSpace(a.messageInput.Text)
	if content == "" {
		return
	}

	if err := a.peer.SendMessage(content); err != nil {
		dialog.ShowError(err, a.mainWindow)
		return
	}

	a.addMessage(a.peer.Username(), content, false)
	a.messageInput.SetText("")
}

func (a *App) sendFile() {
	dialog.ShowFileOpen(func(reader fyne.URIReadCloser, err error) {
		if err != nil || reader == nil {
			return
		}
		defer reader.Close()

		filePath := reader.URI().Path()
		if err := a.peer.SendFile(filePath); err != nil {
			dialog.ShowError(err, a.mainWindow)
			return
		}

		a.addMessage(a.peer.Username(), fmt.Sprintf("📎 Sent file: %s", reader.URI().Name()), true)
	}, a.mainWindow)
}

func (a *App) addMessage(from, content string, isFile bool) {
	a.messages = append(a.messages, chatMessage{
		from:    from,
		content: content,
		isFile:  isFile,
		time:    time.Now(),
	})

	if a.chatHistory != nil && a.currentView == "chat" {
		a.chatHistory.Refresh()
		a.chatHistory.ScrollToBottom()
	}
}
