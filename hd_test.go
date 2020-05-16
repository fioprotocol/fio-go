package fio

import (
	"errors"
	"fmt"
	"testing"
)

func TestHDNewKeys(t *testing.T) {
	mnemonic, err := NewMnemonicFromString("crater husband angle bitter chair rally luggage identify ticket pig toe wear border aerobic wage")
	if err != nil {
		t.Error(err)
		return
	}
	k0 := "5J4s3zFEdkkxTDW7vGvbMFbCnp7Lp2CYKPshdFEqQabPYhiTTZY"
	k1 := "5KhG6QigfDLEDmE5UsHJnYqcHbuEyxDjqmFZBeUgY1sYJpqxqRW"
	k15 := "5J6NKGL4cqbZXfi3fTbXZtPqtDL2wHeoLdmkLg2bnHQF2KSHijs"
	keys, err := mnemonic.Keys(2)
	if err != nil {
		t.Error(err)
		return
	}
	if keys.Keys[0].String() != k0 {
		t.Error("key 0 mismatch")
	}
	if keys.Keys[1].String() != k1 {
		t.Error("key 1 mismatch")
	}
	key, err := mnemonic.KeyAt(15)
	if err != nil {
		t.Error(err)
		return
	}
	// jump a few forward, this should be good enough to prove deterministic derivation
	if key.Keys[0].String() != k15 {
		t.Error("key 15 mismatch")
	}
}

func TestHDGetPubKeys(t *testing.T) {
	mnemonic, err := NewMnemonicFromString("earth dust patient fashion begin behave two brisk solar fetch flash impulse paper around endless")
	if err != nil {
		t.Error(err)
		return
	}
	pk3 := "FIO7KFe37B9FHxRLNGzDA3ACGVY15V6LvVLdohC4ppajUYtwj17KH"
	pk8 := "FIO6qBcB36nBfvbqvmc6xHfucZGQSVJkHHcScvgWvu47oboW2FGxX"
	pk17 := "FIO79wTtYceEozALgxmxQBieRRiK2AiiHL66ssEcNKF49xjbdDWew"
	pubs, err := mnemonic.PubKeys(9)
	if err != nil {
		t.Error(err)
		return
	}
	if pubs[3].String() != pk3 {
		t.Error("public key 3 mismatch")
	}
	if pubs[8].String() != pk8 {
		t.Error("public key 8 mismatch")
	}
	pub, err := mnemonic.PubKeyAt(17)
	if err != nil {
		t.Error(err)
		return
	}
	if pub.String() != pk17 {
		t.Error("public key 17 mismatch")
	}
	// now with 24 words
	mnemonic, err = NewMnemonicFromString("cruise village reflect chunk local dynamic surge verb wave water manage patient clarify speak trick alert throw blood tail between leave special virus donate")
	if err != nil {
		t.Error(err)
		return
	}
	pk3 = "FIO7TBBvXU2QWp5Q3h8T5T7bFhvn1rZUhjtb4g1hw4heHKg5DQUbd"
	pk8 = "FIO5u4s5ddHinq9UhibJ1mL1EzG32855BxEpD48FetKYzFyQc9VSN"
	pk17 = "FIO5bmwWdWooJKzghQkj59R45xLLbPoPPmYGhyk7oujvhcRyjfUFX"
	pubs, err = mnemonic.PubKeys(18)
	if err != nil {
		t.Error(err)
		return
	}
	if pubs[3].String() != pk3 {
		t.Error("public key 3 mismatch")
	}
	if pubs[8].String() != pk8 {
		t.Error("public key 8 mismatch")
	}
	if pubs[17].String() != pk17 {
		t.Error("public key 17 mismatch")
	}
}

func TestMnemonic(t *testing.T) {
	shortMnemonic := "life is too short for debugging javascript"
	longMnemonic := "blah blah blah yah its really long ok get over it we already know this is too long earth dust patient fashion begin behave two brisk solar fetch flash impulse paper around endless"
	mnemonic := "dream knife language movie cannon remove width like wedding gate help patient ocean usage system steak screen summer subway field venture"
	_, err := NewMnemonicFromString(shortMnemonic)
	if err == nil {
		t.Error("allowed too short mnemoic phrase")
	}
	_, err = NewMnemonicFromString(longMnemonic)
	if err == nil {
		t.Error("allowed too long mnemonic phrase")
	}
	mn, err := NewMnemonicFromString(mnemonic)
	if err != nil {
		t.Error(err)
		return
	}
	if mn.Len() != 21 {
		t.Error(errors.New("mnemonic phrase had incorrect length"))
	}
	if mnemonic != mn.String() {
		t.Error("mnemonic did not serialize to string")
	}
}

func TestNewRandomMnemonic(t *testing.T) {
	for _, w := range []int{24, 21, 18, 15, 12} {
		m, err := NewRandomMnemonic(w)
		if err != nil {
			t.Error(err)
		}
		if m.Len() != w {
			t.Error("got wrong word length expecting, got: ", w, m.Len())
		}
		if _, err := m.KeyAt(0); err != nil {
			t.Error("could not derive keys from random mnemonic")
		}
	}
}

func TestMnemonic_Quiz(t *testing.T) {
	mnemonic, err := NewMnemonicFromString("dream knife language movie cannon remove width like wedding gate help patient ocean usage system steak screen summer subway field venture")
	if err != nil {
		t.Error(err)
		return
	}
	q, err := mnemonic.Quiz(0)
	if len(q) != 7 {
		t.Error("didn't get expected number of questions, want 7, got ", len(q))
	}
	for _, quiz := range q {
		if quiz.word != mnemonic.words[quiz.index] {
			t.Error("quiz word was wrong")
		}
		if !quiz.Check(mnemonic.words[quiz.index]) {
			t.Error("quiz question failed")
		}
		if quiz.Check("should fail") {
			t.Error("quiz is returning true for incorrect answer")
		}
	}
	if _, err = mnemonic.Quiz(22); err == nil {
		t.Error("quiz allowed too many questions")
	}
	if n, _ := mnemonic.Quiz(3); n == nil || len(n) != 3 {
		t.Error("didn't get expect count of quiz items")
	}
}

func TestMnemonic_Xpriv(t *testing.T) {
	mnemonic, err := NewMnemonicFromString("struggle dream fetch aunt marriage adult merry machine vessel help slogan bright balcony extend stomach sun father essay surface call song bitter economy approve")
	if err != nil {
		t.Error(err)
		return
	}
	xp, err := mnemonic.Xpriv()
	if err != nil {
		t.Error(err)
		return
	}
	if xp != "xprv9s21ZrQH143K33XDJwNLgRmKTgKXHJuWgq33UMBbrz51vqToDKhysgY6k7SmmmTRkpTjAPpGSb6gKTDZ43WGW3ogozKq4qeENED3pPMcrAr" {
		t.Error("Xpriv did not match")
		fmt.Println(xp)
	}
}

func TestMnemonic_Xpub(t *testing.T) {
	mnemonic, err := NewMnemonicFromString("struggle dream fetch aunt marriage adult merry machine vessel help slogan bright balcony extend stomach sun father essay surface call song bitter economy approve")
	if err != nil {
		t.Error(err)
		return
	}
	xp, err := mnemonic.Xpub()
	if err != nil {
		t.Error(err)
		return
	}
	if xp != "xpub661MyMwAqRbcFXbgQxuM3Zi41iA1gmdN43xeGjbDRKbzodnwks2ERUrabQqhPKag9vHNcZzTz9vYQGLQUubHrtuNmb9faBM7eJQSSGYX7na" {
		t.Error("Xpub did not match")
		fmt.Println(xp)
	}
}