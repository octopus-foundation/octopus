/*
               .'\   /`.
             .'.-.`-'.-.`.
        ..._:   .-. .-.   :_...
      .'    '-.(o ) (o ).-'    `.
     :  _    _ _`~(_)~`_ _    _  :
    :  /:   ' .-=_   _=-. `   ;\  :
    :   :|-.._  '     `  _..-|:   :
     :   `:| |`:-:-.-:-:'| |:'   :
      `.   `.| | | | | | |.'   .'
        `.   `-:_| | |_:-'   .'
          `-._   ````    _.-'
              ``-------''
Created by ab, 21.10.2022
*/

package keys

import (
	"embed"
	"golang.org/x/crypto/ssh"
)

//go:embed *
var AnsibleKeysFS embed.FS

func GetAnsiblePrivateKey() (ssh.AuthMethod, error) {
	keyBytes, err := AnsibleKeysFS.ReadFile("id_rsa")
	if err != nil {
		return nil, err
	}

	privateKey, err := ssh.ParsePrivateKey(keyBytes)
	if err != nil {
		return nil, err
	}

	return ssh.PublicKeys(privateKey), nil
}
