package types

type Edge struct {
	FromSID string
	ToSID   string
	Weight  int
	RID     string
}

type GraphType map[string][]Edge
