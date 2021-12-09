package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/csaf-poc/csaf_distribution/csaf"
	"github.com/csaf-poc/csaf_distribution/util"
)

func ensureFolders(c *config) error {

	wellknown := filepath.Join(c.Web, ".well-known")
	wellknownCSAF := filepath.Join(wellknown, "csaf")

	if err := createWellknown(wellknownCSAF); err != nil {
		return err
	}

	if err := createFeedFolders(c, wellknownCSAF); err != nil {
		return err
	}

	if err := createProviderMetadata(c, wellknownCSAF); err != nil {
		return err
	}

	return createSecurity(c, wellknown)
}

func createWellknown(wellknown string) error {
	st, err := os.Stat(wellknown)
	if err != nil {
		if os.IsNotExist(err) {
			return os.MkdirAll(wellknown, 0755)
		}
		return err
	}
	if !st.IsDir() {
		return errors.New(".well-known/csaf is not a directory")
	}
	return nil
}

func createFeedFolders(c *config, wellknown string) error {
	for _, t := range c.TLPs {
		if t == tlpCSAF {
			continue
		}
		tlpLink := filepath.Join(wellknown, string(t))
		if _, err := filepath.EvalSymlinks(tlpLink); err != nil {
			if os.IsNotExist(err) {
				tlpFolder := filepath.Join(c.Folder, string(t))
				if tlpFolder, err = util.MakeUniqDir(tlpFolder); err != nil {
					return err
				}
				if err = os.Symlink(tlpFolder, tlpLink); err != nil {
					return err
				}
			} else {
				return err
			}
		}
	}
	return nil
}

func createSecurity(c *config, wellknown string) error {
	security := filepath.Join(wellknown, "security.txt")
	if _, err := os.Stat(security); err != nil {
		if os.IsNotExist(err) {
			f, err := os.Create(security)
			if err != nil {
				return err
			}
			fmt.Fprintf(
				f, "CSAF: %s/.well-known/csaf/provider-metadata.json\n",
				c.Domain)
			return f.Close()
		}
		return err
	}
	return nil
}

func createProviderMetadata(c *config, wellknownCSAF string) error {
	path := filepath.Join(wellknownCSAF, "provider-metadata.json")
	_, err := os.Stat(path)
	if err == nil {
		return nil
	}
	if !os.IsNotExist(err) {
		return err
	}
	pm := csaf.NewProviderMetadataDomain(c.Domain, c.modelTLPs())
	pm.Publisher = c.Publisher

	// Set OpenPGP key.
	key, err := c.loadCryptoKey()
	if err != nil {
		return err
	}
	keyID, fingerprint := key.GetHexKeyID(), key.GetFingerprint()
	pm.SetPGP(fingerprint, c.GetOpenPGPURL(keyID))

	return util.WriteToFile(path, pm)
}
