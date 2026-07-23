package query

type GetOffer struct {
	UserID  int64
	OfferID int64
}

type ListOffersByApplication struct {
	HRID          int64
	ApplicationID int64
}

type ListMyOffers struct {
	UserID   int64
	Cursor   string
	PageSize int32
}

type ListOfferEvents struct {
	HRID    int64
	OfferID int64
}
