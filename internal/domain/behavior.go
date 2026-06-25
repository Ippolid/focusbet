package domain

type Behavior struct {
	// AfterFocus: what to do when a session ends.
	// "ask" = prompt, "auto_bank" = silently bank, "auto_rest" = take fair break.
	AfterFocus string `json:"after_focus"`
	// ConfirmEarlyStop: ask before discarding a session's earnings.
	ConfirmEarlyStop bool `json:"confirm_early_stop"`
}
