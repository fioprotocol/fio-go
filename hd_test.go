package fio

import (
	"errors"
	"testing"
)

func TestHDNewKeys(t *testing.T) {
	mnemonic := "crater husband angle bitter chair rally luggage identify ticket pig toe wear border aerobic wage"
	k0 := "5J4s3zFEdkkxTDW7vGvbMFbCnp7Lp2CYKPshdFEqQabPYhiTTZY"
	k1 := "5KhG6QigfDLEDmE5UsHJnYqcHbuEyxDjqmFZBeUgY1sYJpqxqRW"
	k15 := "5J6NKGL4cqbZXfi3fTbXZtPqtDL2wHeoLdmkLg2bnHQF2KSHijs"
	keys, err := HDNewKeys(mnemonic, 16)
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
	// jump a few forward, this should be good enough to prove deterministic derivation
	if keys.Keys[15].String() != k15 {
		t.Error("key 15 mismatch")
	}
}

func TestHDGetPubKeys(t *testing.T) {
	mnemonic := "earth dust patient fashion begin behave two brisk solar fetch flash impulse paper around endless"
	pk3 := "FIO7KFe37B9FHxRLNGzDA3ACGVY15V6LvVLdohC4ppajUYtwj17KH"
	pk8 := "FIO6qBcB36nBfvbqvmc6xHfucZGQSVJkHHcScvgWvu47oboW2FGxX"
	pk17 := "FIO79wTtYceEozALgxmxQBieRRiK2AiiHL66ssEcNKF49xjbdDWew"
	pubs, err := HDGetPubKeys(mnemonic, 18)
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
	// now with 24 words
	mnemonic = "cruise village reflect chunk local dynamic surge verb wave water manage patient clarify speak trick alert throw blood tail between leave special virus donate"
	pk3 = "FIO7TBBvXU2QWp5Q3h8T5T7bFhvn1rZUhjtb4g1hw4heHKg5DQUbd"
	pk8 = "FIO5u4s5ddHinq9UhibJ1mL1EzG32855BxEpD48FetKYzFyQc9VSN"
	pk17 = "FIO5bmwWdWooJKzghQkj59R45xLLbPoPPmYGhyk7oujvhcRyjfUFX"
	pubs, err = HDGetPubKeys(mnemonic, 18)
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
	_, err := MnemonicFromString(shortMnemonic)
	if err == nil {
		t.Error("allowed too short mnemoic phrase")
	}
	_, err = MnemonicFromString(longMnemonic)
	if err == nil {
		t.Error("allowed too long mnemonic phrase")
	}
	mn, err := MnemonicFromString(mnemonic)
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