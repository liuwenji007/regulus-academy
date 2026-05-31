package service

import "errors"

// ErrSessionBusy 会话正在被其他入口处理
var ErrSessionBusy = errors.New("session busy")
