package models

// BA mengembalikan nama Beauty Advisor, atau "" jika tidak diisi.
func (m Mutation) BA() string {
	if m.BAName == nil {
		return ""
	}
	return *m.BAName
}
