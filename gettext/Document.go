package gettext

type Document struct {
	Header  DocumentHeader
	Entries []Entry
}

type DocumentMissingHeaderError struct{}

type DocumentParseError struct {
	Reason string
}

func ParseDocument(s string) (Document, error) {
	return Document{}, nil
}

func (e DocumentMissingHeaderError) Error() string {
	return "The first entry of a document must be a header, having an empty id."
}

func (e DocumentParseError) Error() string {
	return e.Reason
}
