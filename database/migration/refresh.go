package migration

func (self *Migration) Refresh() error {
	if err := self.Reset(); err != nil {
		return err
	}

	return self.Migrate(false)
}
