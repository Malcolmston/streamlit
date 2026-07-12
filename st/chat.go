package st

// ChatMessage adds a chat message bubble attributed to role (commonly "user"
// or "assistant") and returns a container for the message body. Any elements
// may be placed inside it, so a message can contain markdown, tables, charts,
// and so on.
//
//	m := s.ChatMessage("assistant")
//	m.Markdown("Here is your **answer**.")
func (c *Container) ChatMessage(role string) *Container {
	node := c.add("chat_message", props{"role": role})
	return c.child(node)
}

// ChatInput adds a chat entry box pinned to the bottom of its region and
// returns the message the user last submitted, or the empty string before any
// submission. Submitting a message (pressing Enter) reruns the app with the new
// value available. An optional stable key may be supplied.
//
//	if msg := s.ChatInput("Ask me anything"); msg != "" {
//		s.ChatMessage("user").Text(msg)
//	}
func (c *Container) ChatInput(placeholder string, key ...string) string {
	k := c.s.key(optKey(key), "chat_input")
	val := asString(c.s.widgets[k], "")
	c.addWidget(k, "chat_input", props{"placeholder": placeholder, "value": val})
	return val
}
