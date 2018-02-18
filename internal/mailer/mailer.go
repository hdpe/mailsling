package mailer

import "fmt"

type Mailer struct {
	repo   Repository
	client Client
}

func (r *Mailer) ProcessOutstanding() error {
	users, err := r.repo.GetUsersNotWelcomed()

	if err != nil {
		return fmt.Errorf("couldn't get users to be welcomed: %v", err)
	}

	for _, u := range users {
		err = r.client.SubscribeUser(u)
		if err != nil {
			return fmt.Errorf("notify of new user failed: %v", err)
		}

		u.Status = "welcomed"

		err = r.repo.UpdateUser(u)
		if err != nil {
			return fmt.Errorf("couldn't update user: %v", err)
		}
	}

	return nil
}

func NewMailer(repo Repository, client Client) *Mailer {
	return &Mailer{repo, client}
}
