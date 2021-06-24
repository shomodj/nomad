package example

type Basket struct {
	ID    string
	Index int

	Exclude *AppleWithReferenceFields

	ApplePtr  *AppleWithReferenceFields
	BananaPtr *BananaWithOnlyValueFields
	CarrotPtr *CarrotWithCopyMethod
	OrangePtr *OrangeWithEqualsMethod

	AppleStruct  AppleWithReferenceFields
	BananaStruct BananaWithOnlyValueFields
	CarrotStruct CarrotWithCopyMethod
	OrangeStruct OrangeWithEqualsMethod

	AppleMap  map[string]*AppleWithReferenceFields
	BananaMap map[string]*BananaWithOnlyValueFields
	CarrotMap map[string]*CarrotWithCopyMethod
	OrangeMap map[string]*OrangeWithEqualsMethod

	AppleArray  []*AppleWithReferenceFields
	BananaArray []*BananaWithOnlyValueFields
	CarrotArray []*CarrotWithCopyMethod
	OrangeArray []*OrangeWithEqualsMethod
}

type BananaWithOnlyValueFields struct {
	Index      int
	Fig        Fig
	Grapefruit Grapefruit
}

type AppleWithReferenceFields struct {
	Index      int
	Fig        *Fig
	Grapefruit *Grapefruit
}

type CarrotWithCopyMethod struct {
	ID string
}

func (b *CarrotWithCopyMethod) Copy() *CarrotWithCopyMethod { return nil }

type OrangeWithEqualsMethod struct {
	ID string
}

func (b *OrangeWithEqualsMethod) Equals(o *OrangeWithEqualsMethod) bool { return false }

type Fig struct {
	ID string
}

type Grapefruit struct {
	ID string
}
