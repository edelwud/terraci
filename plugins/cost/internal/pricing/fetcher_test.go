package pricing

import "testing"

func TestMatchesLookup_EmptyProductFamily(t *testing.T) {
	price := Price{
		ProductFamily: "Compute Instance",
		Attributes: map[string]string{
			"instanceType": "t3.micro",
		},
	}

	lookup := PriceLookup{
		ProductFamily: "",
		Attributes: map[string]string{
			"instanceType": "t3.micro",
		},
	}

	if !matchesLookup(price, lookup) {
		t.Error("matchesLookup should match when ProductFamily is empty")
	}
}

func TestMatchesLookup_EmptyAttributes(t *testing.T) {
	price := Price{
		ProductFamily: "Compute Instance",
		Attributes: map[string]string{
			"instanceType": "t3.micro",
		},
	}

	lookup := PriceLookup{
		ProductFamily: "Compute Instance",
		Attributes:    map[string]string{},
	}

	if !matchesLookup(price, lookup) {
		t.Error("matchesLookup should match when Attributes is empty")
	}
}

func TestMatchesLookup_MissingAttribute(t *testing.T) {
	price := Price{
		ProductFamily: "Compute Instance",
		Attributes: map[string]string{
			"instanceType": "t3.micro",
		},
	}

	lookup := PriceLookup{
		ProductFamily: "Compute Instance",
		Attributes: map[string]string{
			"nonExistentKey": "value",
		},
	}

	if matchesLookup(price, lookup) {
		t.Error("matchesLookup should not match when required attribute is missing")
	}
}
