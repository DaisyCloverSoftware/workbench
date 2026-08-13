package core

import "sync"

// knowledgeMu protects read-modify-write access to the local JSON knowledge
// store inside one Workbench process. This matters once multiple autonomous
// tasks can retrieve and distill memory concurrently.
var knowledgeMu sync.RWMutex
