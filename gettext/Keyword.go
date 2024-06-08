package gettext

type Keyword struct {
	IsEmpty   bool
	Keyword   string
	Index     int
	IsIndexed bool
}

func SimpleKeyword(keyword string) Keyword {
	return Keyword{Keyword: keyword}
}

func IndexedKeyword(keyword string, index int) Keyword {
	return Keyword{Keyword: keyword, Index: index, IsIndexed: true}
}
