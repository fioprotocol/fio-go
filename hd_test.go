package fio

import (
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
}
