package esmodels

type EmbeddedDocRefWrapper struct {
	EmbeddedDocRefs []EmbeddedDocRef `bson:"embedded_doc_refs"`
}
