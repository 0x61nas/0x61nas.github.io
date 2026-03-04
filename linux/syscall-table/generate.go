package main

import (
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
)

type syscall struct {
	Number      string
	Name        string
	Implemented bool
	Entrypoint  string
	Args        []string
}

type archConfig struct {
	Name      string
	Arch      string
	Title     string
	PageURL   string
	TableURL  string
	TableName string
	Registers []string
	ReturnReg string
	SyscallReg string
	OutFile   string
}

const Version = "7.0-rc2"
const KernelBase = "https://raw.githubusercontent.com/torvalds/linux/v" + Version

func main() {
	resp, err := http.Get(KernelBase + "/include/linux/syscalls.h")
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	headers, _ := io.ReadAll(resp.Body)

	configs := []archConfig{
		{
			Name:      "x86_64",
			Arch:      "x86_64",
			Title:     "Searchable Linux Syscall Table for x86_64",
			PageURL:   "https://thatsillyman.win/linux/syscall-table/",
			TableURL:  KernelBase + "/arch/x86/entry/syscalls/syscall_64.tbl",
			TableName: "syscall_64.tbl",
			Registers: []string{"rdi", "rsi", "rdx", "r10", "r8", "r9"},
			ReturnReg: "%rax",
			SyscallReg: "%rax",
			OutFile:   "x86_64/index.html",
		},
		{
			Name:      "arm64",
			Arch:      "ARM64",
			Title:     "Searchable Linux Syscall Table for ARM64",
			PageURL:   "https://thatsillyman.win/linux/syscall-table/arm64",
			TableURL:  KernelBase + "/include/uapi/asm-generic/unistd.h",
			TableName: "unistd.h",
			Registers: []string{"x0", "x1", "x2", "x3", "x4", "x5"},
			ReturnReg: "%x0",
			SyscallReg: "%x8",
			OutFile:   "arm64/index.html",
		},
		{
			Name:      "riscv",
			Arch:      "RISC-V",
			Title:     "Searchable Linux Syscall Table for RISC-V",
			PageURL:   "https://thatsillyman.win/linux/syscall-table/riscv",
			TableURL:  KernelBase + "/include/uapi/asm-generic/unistd.h",
			TableName: "unistd.h",
			Registers: []string{"a0", "a1", "a2", "a3", "a4", "a5"},
			ReturnReg: "%x1",
			SyscallReg: "%a7",
			OutFile:   "riscv/index.html",
		},
	}

	var wg sync.WaitGroup
	for _, conf := range configs {
		wg.Add(1)
		go func(c archConfig) {
			defer wg.Done()
			generatePage(c, headers)
		}(conf)
	}
	wg.Wait()
}

func generatePage(conf archConfig, headers []byte) {
	syscalls := fetchSyscalls(conf, headers)
	tmpl, err := template.ParseFiles("index.html.tmpl")
	if err != nil {
		log.Printf("[%s] Template error: %v", conf.Name, err)
		return
	}

	f, err := os.Create(conf.OutFile)
	if err != nil {
		log.Printf("[%s] File error: %v", conf.Name, err)
		return
	}
	defer f.Close()

	tmpl.Execute(f, map[string]interface{}{
		"Title": conf.Title,
		"Arch": conf.Arch, 
		"PageURL": conf.PageURL,
		"TableURL": conf.TableURL,
		"TableName": conf.TableName,
		"Version":   Version,
		"Syscalls":  syscalls,
		"Registers": conf.Registers,
		"ReturnReg": conf.ReturnReg,
		"SyscallReg": conf.SyscallReg,
	})
	log.Printf("Generated %s", conf.OutFile)
}

func fetchSyscalls(conf archConfig, headers []byte) []syscall {
	resp, err := http.Get(conf.TableURL)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	table, _ := io.ReadAll(resp.Body)

	var syscalls []syscall
	lines := strings.Split(string(table), "\n")

	if strings.HasSuffix(conf.TableURL, ".tbl") {
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			parts := strings.Fields(line)
			if len(parts) < 3 {
				continue
			}
			if len(parts) > 1 && parts[1] == "x32" {
				continue
			}

			entry := ""
			if len(parts) >= 4 {
				entry = parts[3]
			}
			syscalls = append(syscalls, parseEntry(parts[0], parts[2], entry, headers))
		}
	} else {
		reGeneric := regexp.MustCompile(`#define __NR_(\w+)\s+(\d+)`)
		for _, line := range lines {
			m := reGeneric.FindStringSubmatch(line)
			if m != nil {
				name := m[1]
				num := m[2]
				syscalls = append(syscalls, parseEntry(num, name, "sys_"+name, headers))
			}
		}
	}
	return syscalls
}

func parseEntry(num, name, entry string, headers []byte) syscall {
	if entry == "" {
		return syscall{Number: num, Name: name, Implemented: false}
	}
	
	if entry == "sys_mmap" {
		entry = "ksys_mmap_pgoff"
	}

	re := regexp.MustCompile(`(?:asmlinkage|unsigned) long ` + entry + `\(([^)]+)\);`)
	matches := re.FindStringSubmatch(string(headers))
	
	var args []string
	if matches != nil {
		if matches[1] == "void" {
			args = []string{}
		} else {
			args = strings.Split(matches[1], ",")
			for i, arg := range args {
				args[i] = strings.TrimSpace(strings.ReplaceAll(arg, "__user ", ""))
			}
		}
	} else {
		switch entry {
		case "sys_rt_sigreturn": args = []string{}
		case "sys_modify_ldt": args = []string{"int func", "void *ptr", "unsigned long bytecount"}
		case "sys_arch_prctl": args = []string{"int option", "unsigned long arg2"}
		case "sys_iopl": args = []string{"unsigned int level"}
		default: args = []string{"complex arguments"}
		}
	}

	return syscall{
		Number:      num,
		Name:        name,
		Implemented: true,
		Entrypoint:  strings.TrimPrefix(entry, "sys_"),
		Args:        args,
	}
}
