package sample

type Account struct{}

func NewAccount() *Account {
	return &Account{}
}

func (a *Account) Credit(amount int) int {
	if amount > 0 {
		return amount
	}
	return 0
}
