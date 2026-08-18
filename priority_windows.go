package main

import "golang.org/x/sys/windows"

// lowerProcessPriority coloca o processo em modo de segundo plano do
// Windows: reduz automaticamente a prioridade de CPU, de I/O de disco e de
// memória, para que a varredura de arquivos não compita por recursos com
// o que o usuário estiver usando ativamente.
func lowerProcessPriority() {
	handle := windows.CurrentProcess()
	_ = windows.SetPriorityClass(handle, windows.PROCESS_MODE_BACKGROUND_BEGIN)
}
