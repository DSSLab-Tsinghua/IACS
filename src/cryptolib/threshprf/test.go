// test
package threshprf

import (
	"fmt"
)

func Test_course() {

	n := 5
	k := 3
	Init_key_dealer(int64(n), int64(k))
	C := []byte("0123456789abcdef")

	id_1 := 1
	skb, vkx, vky := LoadkeyFromFiles(int64(id_1))
	share := Compute_share(C, skb, vkx, vky)

	if Verify_share(C, vkx, vky, share) == true {
		Store_share(share, int64(id_1))
	} else {
		fmt.Printf("\n share for party 1 is false")
	}
	//====================================================

	id_2 := 2
	skb, vkx, vky = LoadkeyFromFiles(int64(id_2))
	share = Compute_share(C, skb, vkx, vky)

	if Verify_share(C, vkx, vky, share) == true {
		Store_share(share, int64(id_2))
	} else {
		fmt.Printf("\n share for party 2 is false")
	}
	//====================================================

	id_3 := 3
	skb, vkx, vky = LoadkeyFromFiles(int64(id_3))
	share = Compute_share(C, skb, vkx, vky)

	if Verify_share(C, vkx, vky, share) == true {
		Store_share(share, int64(id_3))
	} else {
		fmt.Printf("\n share for party 3 is false")
	}
	//====================================================

	idarr := []int64{int64(id_3), int64(id_2), int64(id_1)}
	prf := Compute_prf(idarr, int64(k))
	fmt.Printf("\n prf is %x", prf)
	//===================================================

	id_0 := 0
	skb, vkx, vky = LoadkeyFromFiles(int64(id_0))
	share = Compute_share(C, skb, vkx, vky)
	fmt.Printf("\n share_0 is %x", share[0:64])
	//====================================================

}

func Test_course_10() {

	n := 10
	k := 6
	Init_key_dealer(int64(n), int64(k))
	C := []byte("0123456789abcdef")

	id_1 := 1
	skb, vkx, vky := LoadkeyFromFiles(int64(id_1))
	share := Compute_share(C, skb, vkx, vky)

	if Verify_share(C, vkx, vky, share) == true {
		Store_share(share, int64(id_1))
	} else {
		fmt.Printf("\n share for party 1 is false")
	}
	//====================================================

	id_2 := 2
	skb, vkx, vky = LoadkeyFromFiles(int64(id_2))
	share = Compute_share(C, skb, vkx, vky)

	if Verify_share(C, vkx, vky, share) == true {
		Store_share(share, int64(id_2))
	} else {
		fmt.Printf("\n share for party 2 is false")
	}
	//====================================================

	id_3 := 3
	skb, vkx, vky = LoadkeyFromFiles(int64(id_3))
	share = Compute_share(C, skb, vkx, vky)

	if Verify_share(C, vkx, vky, share) == true {
		Store_share(share, int64(id_3))
	} else {
		fmt.Printf("\n share for party 3 is false")
	}
	//====================================================

	id_4 := 4
	skb, vkx, vky = LoadkeyFromFiles(int64(id_4))
	share = Compute_share(C, skb, vkx, vky)

	if Verify_share(C, vkx, vky, share) == true {
		Store_share(share, int64(id_4))
	} else {
		fmt.Printf("\n share for party 4 is false")
	}
	//====================================================

	id_5 := 6
	skb, vkx, vky = LoadkeyFromFiles(int64(id_5))
	share = Compute_share(C, skb, vkx, vky)

	if Verify_share(C, vkx, vky, share) == true {
		Store_share(share, int64(id_5))
	} else {
		fmt.Printf("\n share for party 5 is false")
	}
	//====================================================

	id_6 := 7
	skb, vkx, vky = LoadkeyFromFiles(int64(id_6))
	share = Compute_share(C, skb, vkx, vky)

	if Verify_share(C, vkx, vky, share) == true {
		Store_share(share, int64(id_6))
	} else {
		fmt.Printf("\n share for party 6 is false")
	}

	id_7 := 8
	skb, vkx, vky = LoadkeyFromFiles(int64(id_7))
	share = Compute_share(C, skb, vkx, vky)

	if Verify_share(C, vkx, vky, share) == true {
		Store_share(share, int64(id_7))
	} else {
		fmt.Printf("\n share for party 6 is false")
	}
	//====================================================

	idarr := []int64{int64(id_1), int64(id_2), int64(id_3), int64(id_4), int64(id_5), int64(id_6)}
	prf := Compute_prf(idarr, int64(k))
	fmt.Printf("\n prf is %x", prf)
	//===================================================

	idarr = []int64{int64(id_2), int64(id_7), int64(id_1), int64(id_4), int64(id_5), int64(id_6)}
	prf1 := Compute_prf(idarr, int64(k))
	fmt.Printf("\n prf1 is %x", prf1)

	id_0 := 0
	skb, vkx, vky = LoadkeyFromFiles(int64(id_0))
	share = Compute_share(C, skb, vkx, vky)
	fmt.Printf("\n share_0 is %x", share[0:64])
	//====================================================

}
