package telegram

// Telegram Bot API types — only what visor needs, not the full spec.

type Update struct {
	UpdateID          int                `json:"update_id"`
	Message           *Message           `json:"message,omitempty"`
	MessageReaction   *MessageReaction   `json:"message_reaction,omitempty"`
}

type Message struct {
	MessageID int    `json:"message_id"`
	From      *User  `json:"from,omitempty"`
	Chat      Chat   `json:"chat"`
	Date      int    `json:"date"`
	Text      string `json:"text,omitempty"`
	Voice     *Voice `json:"voice,omitempty"`
	Photo     []PhotoSize `json:"photo,omitempty"`
	Caption   string `json:"caption,omitempty"`
}

type User struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
	Username  string `json:"username,omitempty"`
}

type Chat struct {
	ID   int64  `json:"id"`
	Type string `json:"type"`
}

type Voice struct {
	FileID   string `json:"file_id"`
	Duration int    `json:"duration"`
}

type PhotoSize struct {
	FileID   string `json:"file_id"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	FileSize int    `json:"file_size,omitempty"`
}

type MessageReaction struct {
	Chat        Chat   `json:"chat"`
	MessageID   int    `json:"message_id"`
	User        *User  `json:"user,omitempty"`
	NewReaction []Reaction `json:"new_reaction,omitempty"`
}

type Reaction struct {
	Type  string `json:"type"`
	Emoji string `json:"emoji,omitempty"`
}
