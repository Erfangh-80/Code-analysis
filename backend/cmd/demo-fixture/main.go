// demo-fixture creates a local-only Git repository used by the demo and tests.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func main() {
	output := flag.String("path", "./demo-repository", "destination for the new demo Git repository")
	force := flag.Bool("force", false, "explicitly remove an existing destination before creating the fixture")
	flag.Parse()
	if info, err := os.Stat(*output); err == nil && info.IsDir() {
		if !*force {
			fmt.Fprintln(os.Stderr, "refusing to overwrite existing directory; choose a new --path or pass --force explicitly")
			os.Exit(2)
		}
		if err := os.RemoveAll(*output); err != nil {
			fatal(err)
		}
	}
	if err := os.MkdirAll(*output, 0755); err != nil {
		fatal(err)
	}
	git(*output, nil, "init")
	write(*output, "main.go", `package demo

func Discount(total int, member bool) int {
	if member && total > 100 { return total - 20 }
	return total
}
`)
	write(*output, "src/payment.py", `class PaymentService:
    def charge(self, amount, verified):
        if not verified:
            return "denied"
        for _ in range(amount):
            if amount > 3 and verified:
                return "review"
        return "ok"
`)
	write(*output, "web/cart.js", `export function total(items) {
  let sum = 0
  for (const item of items) {
    if (item.active) sum += item.price
  }
  return sum
}
`)
	commit(*output, "2025-01-05T09:00:00Z", "Ada", "ada@example.test", "initial implementation")
	write(*output, "src/payment.py", `class PaymentService:
    def charge(self, amount, verified):
        if not verified:
            return "denied"
        for _ in range(amount):
            if amount > 3 and verified:
                return "review"
        return "ok"

    def refund(self, status):
        return "queued" if status else "ignored"
`)
	commit(*output, "2025-01-15T09:00:00Z", "Ben", "ben@example.test", "add refund workflow")
	historical := trim(git(*output, nil, "rev-parse", "HEAD"))
	write(*output, "src/payment.py", `class PaymentService:
    def charge(self, amount, verified):
        if not verified:
            return "denied"
        for _ in range(amount):
            if amount > 3 and verified:
                return "review"
        return "ok"

    def refund(self, status):
        if status == "paid":
            return "queued"
        return "ignored"
`)
	commit(*output, "2025-02-02T09:00:00Z", "Ada", "ada@example.test", "fix refund state bug")
	write(*output, "web/cart.js", `export function total(items, coupon) {
  let sum = 0
  for (const item of items) {
    if (item.active && item.price > 0) sum += item.price
  }
  if (coupon) return Math.max(0, sum - coupon)
  return sum
}
`)
	commit(*output, "2025-02-10T09:00:00Z", "Cara", "cara@example.test", "resolve cart calculation edge case")
	fmt.Printf("Repository path: %s\nHistorical commit SHA: %s\nSuggested analysis time range: 2025-01-01T00:00:00Z to 2025-02-01T00:00:00Z\nSuggested evaluation time range: 2025-01-16T00:00:00Z to 2025-03-01T00:00:00Z\n", *output, historical)
}
func write(root, name, content string) {
	full := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
		fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0644); err != nil {
		fatal(err)
	}
}
func commit(root, date, name, email, message string) {
	environment := append(os.Environ(), "GIT_AUTHOR_NAME="+name, "GIT_AUTHOR_EMAIL="+email, "GIT_COMMITTER_NAME="+name, "GIT_COMMITTER_EMAIL="+email, "GIT_AUTHOR_DATE="+date, "GIT_COMMITTER_DATE="+date)
	git(root, environment, "add", ".")
	git(root, environment, "commit", "-m", message)
}
func git(root string, environment []string, args ...string) string {
	command := exec.Command("git", append([]string{"-C", root}, args...)...)
	if environment != nil {
		command.Env = environment
	}
	out, err := command.CombinedOutput()
	if err != nil {
		fatal(fmt.Errorf("git %v: %w: %s", args, err, out))
	}
	return string(out)
}
func trim(value string) string {
	for len(value) > 0 && (value[len(value)-1] == '\n' || value[len(value)-1] == '\r') {
		value = value[:len(value)-1]
	}
	return value
}
func fatal(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
