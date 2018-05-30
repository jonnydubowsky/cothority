// This is a command line interface for communicating with the evoting service.
package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/dedis/cothority"
	"github.com/dedis/cothority/evoting"
	"github.com/dedis/cothority/evoting/lib"
	"github.com/dedis/cothority/skipchain"
	"github.com/dedis/kyber"
	"github.com/dedis/kyber/util/key"
	"github.com/dedis/onet"
	"github.com/dedis/onet/app"
)

var (
	argRoster = flag.String("roster", "", "path to roster toml file")
	argAdmins = flag.String("admins", "", "list of admin users")
	argPin    = flag.String("pin", "", "service pin")
	argKey    = flag.String("key", "", "public key of authentication server")
	argID     = flag.String("id", "", "ID of the master chain to modify (optional)")
	argUser   = flag.Int("user", 0, "The SCIPER of an existing admin of this chain")
	argSig    = flag.String("sig", "", "A signature proving that you can login to Tequila with the given SCIPER.")
	argShow   = flag.Bool("show", false, "Show the current Master config")
	argElec   = flag.String("election", "", "Load an election file.")
)

var days = 24 * time.Hour

func main() {
	flag.Parse()

	if *argRoster == "" {
		log.Fatal("Roster argument (-roster) is required.")
	}
	roster, err := parseRoster(*argRoster)
	if err != nil {
		log.Fatal("cannot parse roster: ", err)
	}

	if *argShow {
		id, err := hex.DecodeString(*argID)
		if err != nil {
			log.Fatal("id decode", err)
		}
		request := &evoting.GetElections{Master: id}
		reply := &evoting.GetElectionsReply{}
		client := onet.NewClient(cothority.Suite, evoting.ServiceName)
		if err = client.SendProtobuf(roster.List[0], request, reply); err != nil {
			log.Fatal("get elections request: ", err)
		}
		m := reply.Master
		fmt.Printf(" Admins: %v\n", m.Admins)
		fmt.Printf(" Roster: %v\n", m.Roster.List)
		fmt.Printf("    Key: %v\n", m.Key)
		return
	}

	if *argElec != "" {
		elec := &lib.Election{
			Name: map[string]string{
				"en": "Jeff's big election.",
				"fr": "Le scrutin geant de Jeff",
			},
			Creator: 289938,
			// the list of voters
			Users:      []uint32{0, 1, 2, 289938},
			Candidates: []uint32{289938},
			MaxChoices: 1,
			Subtitle: map[string]string{
				"en": "Vote here now!",
			},
			MoreInfo: "",
			Start:    time.Now().Unix(),
			End:      time.Now().Add(7 * days).Unix(),
			// Theme is one of:
			// { name: 'EPFL', class: 'epfl' },
			// { name: 'ENAC', class: 'enac' },
			// { name: 'SB', class: 'sb' },
			// { name: 'STI', class: 'sti' },
			// { name: 'IC', class: 'ic' },
			// { name: 'SV', class: 'sv' },
			// { name: 'CDM', class: 'cdm' },
			// { name: 'CDH', class: 'cdh' },
			// { name: 'INTER', class: 'inter' },
			// { name: 'Associations', class: 'assoc' }
			Theme:  "epfl",
			Footer: lib.Footer{},
		}
		/*
			_, err := toml.DecodeFile(*argElec, elec)
			if err != nil {
				log.Fatal(err)
			}
			log.Print(elec)
			return
		*/

		if *argID == "" {
			log.Fatal("-id required when opening elections")
		}
		id, err := hex.DecodeString(*argID)
		if err != nil {
			log.Fatal("id decode", err)
		}
		request := &evoting.Open{
			ID:       id,
			Election: elec,
		}

		if *argSig == "" {
			log.Fatal("-sig required when opening elections")
		}
		sig, err := hex.DecodeString(*argSig)
		if err != nil {
			log.Fatal("sig decode", err)
		}
		if *argUser == 0 {
			log.Fatal("-user required when opening elections")
		}
		var u = uint32(*argUser)
		request.User = u
		request.Signature = sig

		reply := &evoting.OpenReply{}
		client := onet.NewClient(cothority.Suite, evoting.ServiceName)
		if err = client.SendProtobuf(roster.List[0], request, reply); err != nil {
			log.Fatal("open election request: ", err)
		}
		return
	}

	log.Print("Trying to create a new master chain.")

	if *argAdmins == "" {
		log.Fatal("Admin list (-admins) must have at least one id.")
	}

	admins, err := parseAdmins(*argAdmins)
	if err != nil {
		log.Fatal("cannot parse admins: ", err)
	}

	if *argPin == "" {
		log.Fatal("pin must be set for create and update operations.")
	}

	var pub kyber.Point
	if *argKey != "" {
		pub, err = parseKey(*argKey)
		if err != nil {
			log.Fatal("cannot parse key: ", err)
		}
	} else {
		kp := key.NewKeyPair(cothority.Suite)
		log.Printf("Auth-server private key: %v", kp.Private)
		pub = kp.Public
	}

	request := &evoting.Link{Pin: *argPin, Roster: roster, Key: pub, Admins: admins}
	if *argID != "" {
		id, err := hex.DecodeString(*argID)
		if err != nil {
			log.Fatal("id decode", err)
		}

		if *argSig == "" {
			log.Fatal("-sig required when updating")
		}
		sig, err := hex.DecodeString(*argSig)
		if err != nil {
			log.Fatal("sig decode", err)
		}
		var sbid skipchain.SkipBlockID = id
		request.ID = &sbid
		var u = uint32(*argUser)
		request.User = &u
		request.Signature = &sig
	}
	reply := &evoting.LinkReply{}

	client := onet.NewClient(cothority.Suite, evoting.ServiceName)
	if err = client.SendProtobuf(roster.List[0], request, reply); err != nil {
		log.Fatal("link request: ", err)
	}

	log.Printf("Auth-server public  key: %v", pub)
	log.Printf("Master ID: %x", reply.ID)
}

// parseRoster reads a Dedis group toml file a converts it to a cothority roster.
func parseRoster(path string) (*onet.Roster, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	group, err := app.ReadGroupDescToml(file)
	if err != nil {
		return nil, err
	}
	return group.Roster, nil
}

// parseAdmins converts a string of comma-separated sciper numbers in
// the format sciper1,sciper2,sciper3 to a list of integers.
func parseAdmins(scipers string) ([]uint32, error) {
	if scipers == "" {
		return nil, nil
	}

	admins := make([]uint32, 0)
	for _, admin := range strings.Split(scipers, ",") {
		sciper, err := strconv.Atoi(admin)
		if err != nil {
			return nil, err
		}
		admins = append(admins, uint32(sciper))
	}
	return admins, nil
}

// parseKey unmarshals a Ed25519 point given in hexadecimal form.
func parseKey(key string) (kyber.Point, error) {
	b, err := hex.DecodeString(key)
	if err != nil {
		return nil, err
	}

	point := cothority.Suite.Point()
	if err = point.UnmarshalBinary(b); err != nil {
		return nil, err
	}
	return point, nil
}
