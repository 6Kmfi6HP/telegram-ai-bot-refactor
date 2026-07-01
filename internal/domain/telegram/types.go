package telegram

import "encoding/json"

// PhotoSize is the Telegram Bot API representation of a photo variant.
type PhotoSize struct {
	FileID   string `json:"file_id"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	FileSize int    `json:"file_size,omitempty"`
}

// File is Telegram Bot API file metadata.
type File struct {
	FileID   string `json:"file_id"`
	FilePath string `json:"file_path"`
	FileSize int    `json:"file_size,omitempty"`
}

// InlineKeyboardButton is a Telegram inline keyboard button.
type InlineKeyboardButton struct {
	Text         string `json:"text"`
	CallbackData string `json:"callback_data,omitempty"`
}

// InlineKeyboardMarkup is a Telegram inline keyboard layout.
type InlineKeyboardMarkup struct {
	InlineKeyboard [][]InlineKeyboardButton `json:"inline_keyboard"`
}

// Update is the Telegram Bot API update payload used by the bot.
type Update struct {
	UpdateID      int            `json:"update_id"`
	Message       *Message       `json:"message"`
	CallbackQuery *CallbackQuery `json:"callback_query"`
}

// CallbackQuery is a Telegram callback query from inline keyboards.
type CallbackQuery struct {
	ID      string   `json:"id"`
	From    *User    `json:"from"`
	Message *Message `json:"message"`
	Data    string   `json:"data"`
}

// Message is the subset of Telegram messages required by the bot.
type Message struct {
	MessageID      int         `json:"message_id"`
	Text           string      `json:"text"`
	Caption        string      `json:"caption,omitempty"`
	Photo          []PhotoSize `json:"photo,omitempty"`
	Date           int         `json:"date"`
	Chat           *Chat       `json:"chat"`
	From           *User       `json:"from"`
	ReplyToMessage *Message    `json:"reply_to_message"`
}

// Chat is the Telegram chat identifier wrapper.
type Chat struct {
	ID int `json:"id"`
}

// User is the Telegram user representation used by the bot.
type User struct {
	ID        int    `json:"id"`
	UserName  string `json:"username"`
	FirstName string `json:"first_name"`
}

// APIResponse is the standard Telegram Bot API response envelope.
type APIResponse struct {
	Ok          bool            `json:"ok"`
	Result      json.RawMessage `json:"result"`
	ErrorCode   int             `json:"error_code"`
	Description string          `json:"description"`
}

// BotCommand is a Telegram command menu item.
type BotCommand struct {
	Command     string `json:"command"`
	Description string `json:"description"`
}
