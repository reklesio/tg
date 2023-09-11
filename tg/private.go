package tg

import (
	"fmt"

	//tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type context struct {
	Session *Session
	// To reach the bot abilities inside callbacks.
	Bot     *Bot
	skippedUpdates chan *Update
	// Current screen ID.
	screenId, prevScreenId ScreenId
}


// The type represents way to interact with user in
// handling functions. Is provided to Act() function always.

// Goroutie function to handle each user.
func (c *context) handleUpdateChan(updates chan *Update) {
	beh := c.Bot.behaviour
	if beh.Init != nil {
		c.run(beh.Init, nil)
	}
	beh.Root.Serve(&Context{
		context: c,
	}, updates)
}


func (c *context) run(a Action, u *Update) {
	a.Act(&Context{context: c, Update:  u})
}

func (c *Context) ScreenId() ScreenId {
	return c.screenId
}

func (c *Context) PrevScreenId() ScreenId {
	return c.prevScreenId
}

func (c *Context) Run(a Action, u *Update) {
	if a != nil {
		a.Act(&Context{context: c.context, Update: u})
	}
}

// Only for the root widget usage.
// Skip the update sending it down to
// the underlying widget.
func (c *Context) Skip(u *Update) {
	if c.skippedUpdates != nil {
		c.skippedUpdates <- u
	}
}

// Renders the Renedrable object to the side of client
// and returns the messages it sent.
func (c *Context) Render(v Renderable) ([]*Message, error) {
	return c.Bot.Render(c.Session.Id, v)
}

// Sends to the Sendable object.
func (c *Context) Send(v Sendable) (*Message, error) {
	return c.Bot.Send(c.Session.Id, v)
}

// Sends the formatted with fmt.Sprintf message to the user.
func (c *Context) Sendf(format string, v ...any) (*Message, error) {
	msg, err := c.Send(NewMessage(
		c.Session.Id, fmt.Sprintf(format, v...),
	))
	if err != nil {
		return nil, err
	}
	return msg, err
}

// Interface to interact with the user.
type Context struct {
	*context
	// The update that called the Context usage.
	*Update
	// Used as way to provide outer values redirection
	// into widgets and actions 
	Arg any
}

// Customized actions for the bot.
type Action interface {
	Act(*Context)
}

type ActionFunc func(*Context)

func (af ActionFunc) Act(c *Context) {
	af(c)
}

// The type implements changing screen to the underlying ScreenId
type ScreenChange ScreenId

func (sc ScreenChange) Act(c *Context) {
	if !c.Bot.behaviour.ScreenExist(ScreenId(sc)) {
		panic(ScreenNotExistErr)
	}
	err := c.ChangeScreen(ScreenId(sc))
	if err != nil {
		panic(err)
	}
}

type C = Context

// Changes screen of user to the Id one.
func (c *Context) ChangeScreen(screenId ScreenId) error {
	if !c.Bot.behaviour.ScreenExist(screenId) {
		return ScreenNotExistErr
	}

	// Stop the reading by sending the nil,
	// since we change the screen and
	// current goroutine needs to be stopped.
	// if c.readingUpdate {
		// c.Updates <- nil
	// }

	// Getting the screen and changing to
	// then executing its widget.
	screen := c.Bot.behaviour.Screens[screenId]
	c.prevScreenId = c.screenId
	c.screenId = screenId

	// Making the new channel for the widget.
	if c.skippedUpdates != nil {
		close(c.skippedUpdates)
	}
	c.skippedUpdates = make(chan *Update)
	if screen.Widget != nil {
		// Running the widget if the screen has one.
		go func() {
			screen.Widget.Serve(c, c.skippedUpdates)
		}()
	} else {
		panic("no widget defined for the screen")
	}

	return nil
}
