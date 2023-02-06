package fermyon

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/fermyon/actions/fermyon-cloud-login/pkg/uidriver"
	"github.com/sirupsen/logrus"
	"github.com/tebeka/selenium"
	"github.com/xlzd/gotp"
)

type Token struct {
	Token      string    `json:"token"`
	Expiration time.Time `json:"expiration"`
}

func LoginWithGithub(cloudLink string, username, password string) (string, error) {
	ui, err := uidriver.New()
	if err != nil {
		return "", fmt.Errorf("connecting to selenium: %w", err)
	}

	defer func(ui *uidriver.Driver) {
		screenshot, err := ui.WebDriver.Screenshot()
		if err != nil {
			logrus.Warnf("capturing screenshot: %v", err)
		}

		err = os.WriteFile("screenshot.png", screenshot, 0644)
		if err != nil {
			logrus.Warnf("saving screenshot: %v", err)
		}

		ui.WebDriver.Close()
		ui.WebDriver.Quit()
	}(ui)

	logrus.Infof("opening Fermyon cloud at %s", cloudLink)
	err = ui.WebDriver.Get(cloudLink)
	if err != nil {
		return "", err
	}

	logrus.Infof("clicking on login with github")
	el, err := ui.WebDriver.FindElement(selenium.ByXPATH, "//button/span[text()='Login with GitHub']")
	if err != nil {
		return "", err
	}

	err = el.Click()
	if err != nil {
		return "", err
	}

	logrus.Infof("Entering creds on github login page")
	el, err = ui.WebDriver.FindElement(selenium.ByID, "login_field")
	if err != nil {
		return "", err
	}

	err = el.SendKeys(username)
	if err != nil {
		return "", err
	}

	el, err = ui.WebDriver.FindElement(selenium.ByID, "password")
	if err != nil {
		return "", err
	}

	err = el.SendKeys(password)
	if err != nil {
		return "", err
	}

	el, err = ui.WebDriver.FindElement(selenium.ByName, "commit")
	if err != nil {
		return "", err
	}

	err = el.Click()
	if err != nil {
		return "", err
	}

	logrus.Infof("handling diff auth challenges offered by Github")
	err = handle2FA(ui)
	if err != nil {
		return "", err
	}
	logrus.Infof("entered 2fa")

	var rawToken string
	ui.WebDriver.WaitWithTimeoutAndInterval(func(driver selenium.WebDriver) (bool, error) {
		url, err := driver.CurrentURL()
		if err != nil {
			logrus.Debugf(err.Error())
			return false, nil
		}

		if !strings.Contains(url, cloudLink) {
			logrus.Debugf("waiting for url to be %s, current url %s\n", cloudLink, url)
			return false, nil
		}

		logrus.Debugf("current url is %s\n", url)
		raw, err := driver.ExecuteScript("return localStorage.getItem('token');", nil)
		if err != nil {
			logrus.Debugf(err.Error())
			return false, nil
		}

		if rawStr, ok := raw.(string); ok && rawStr != "" {
			rawToken = rawStr
			return true, nil
		}

		logrus.Infof("waiting for login to cloud to finish\n")
		return false, nil
	}, 30*time.Second, 1*time.Second)

	token := &Token{}
	err = json.Unmarshal([]byte(rawToken), token)
	if err != nil {
		return "", err
	}

	return token.Token, nil
}

func handle2FA(ui *uidriver.Driver) error {
	el, err := ui.WebDriver.FindElement(selenium.ByID, "totp")
	if err != nil {
		return err
	}

	otp := gotp.NewDefaultTOTP(os.Getenv("E2E_GH_TOTP_SECRET")).Now()
	err = el.SendKeys(otp)
	if err != nil {
		return err
	}
	return nil
}
