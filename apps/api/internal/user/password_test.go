package user

import "testing"

func TestHashAndCompareRoundTrip(t *testing.T) {
	h := NewBcryptHasher(4)

	hash, err := h.Hash("correct-horse")
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}
	if hash == "correct-horse" {
		t.Fatal("Hash returned the plaintext")
	}
	if err := h.Compare(hash, "correct-horse"); err != nil {
		t.Errorf("Compare with the right password failed: %v", err)
	}
}

func TestCompareRejectsWrongPassword(t *testing.T) {
	h := NewBcryptHasher(4)
	hash, _ := h.Hash("correct-horse")

	if err := h.Compare(hash, "battery-staple"); err == nil {
		t.Error("Compare accepted the wrong password")
	}
}

// bcrypt salts every hash, so the same password must not produce a stable
// digest - otherwise identical passwords are visibly identical in the DB.
func TestHashIsSaltedPerCall(t *testing.T) {
	h := NewBcryptHasher(4)

	first, _ := h.Hash("same-password")
	second, _ := h.Hash("same-password")

	if first == second {
		t.Error("hashing the same password twice produced identical output - not salted")
	}
	if err := h.Compare(second, "same-password"); err != nil {
		t.Errorf("the second hash does not verify: %v", err)
	}
}

func TestNewBcryptHasherDefaultsNonPositiveCost(t *testing.T) {
	for _, cost := range []int{0, -1} {
		h := NewBcryptHasher(cost)
		hash, err := h.Hash("whatever")
		if err != nil {
			t.Fatalf("cost %d: Hash failed: %v", cost, err)
		}
		if err := h.Compare(hash, "whatever"); err != nil {
			t.Errorf("cost %d: hash does not verify: %v", cost, err)
		}
	}
}
