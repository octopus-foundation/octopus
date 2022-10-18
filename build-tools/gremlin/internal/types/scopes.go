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

Created by ab, 29.09.2022
*/

package types

import "strings"

type ScopedName struct {
	name     string
	parent   []string
	fullPath string

	localPath []string

	platformName map[TargetPlatform]string
}

func ParseName(name string) ScopedName {
	parts := strings.Split(name, ".")
	if len(parts) > 1 {
		return ScopedName{
			name:     parts[len(parts)-1],
			parent:   parts[:len(parts)-1],
			fullPath: name,
		}
	} else {
		return ScopedName{
			name:     name,
			parent:   nil,
			fullPath: name,
		}
	}
}

func (s ScopedName) Child(name string) ScopedName {
	var path []string
	path = append(path, s.parent...)
	if s.name != "" {
		path = append(path, s.name)
	}
	path = append(path, name)

	return ScopedName{
		name:     name,
		parent:   path[:len(path)-1],
		fullPath: strings.Join(path, "."),
	}
}

func (s ScopedName) LocalChild(name string) ScopedName {
	child := s.Child(name)

	var localPath []string
	localPath = append(localPath, s.localPath...)
	localPath = append(localPath, s.name)

	child.localPath = localPath
	return child
}

func (s ScopedName) ToScope(target ScopedName) ScopedName {
	if target.fullPath == "" {
		return s
	}

	var path []string
	path = append(path, target.parent...)
	if target.name != "" {
		path = append(path, target.name)
	}
	path = append(path, s.parent...)
	path = append(path, s.name)

	return ScopedName{
		name:     s.name,
		parent:   path[:len(path)-1],
		fullPath: strings.Join(path, "."),
	}
}

func (s ScopedName) ToParent() ScopedName {
	if len(s.parent) == 0 {
		return ScopedName{}
	}
	var path []string
	path = append(path, s.parent...)

	return ScopedName{
		name:         path[len(path)-1],
		parent:       path[:len(path)-1],
		fullPath:     strings.Join(path, "."),
		platformName: map[TargetPlatform]string{},
	}
}

func (s ScopedName) Equal(partialName ScopedName) bool {
	if partialName.name != s.name {
		return false
	}

	return s.fullPath == partialName.fullPath
}

func (s ScopedName) IsIn(parent ScopedName) bool {
	if len(s.parent) <= len(parent.parent) {
		return false
	}

	if len(parent.parent) == 0 {
		return true
	}

	for i, p := range parent.parent {
		if s.parent[i] != p {
			return false
		}
	}

	return s.parent[len(parent.parent)] == parent.name
}

func (s ScopedName) PlatformName(platform TargetPlatform) string {
	if s.platformName == nil {
		return ""
	}
	return s.platformName[platform]
}

func (s ScopedName) WithPlatformName(platform TargetPlatform, name string) ScopedName {
	platforms := map[TargetPlatform]string{}
	for k, v := range s.platformName {
		platforms[k] = v
	}
	platforms[platform] = name
	var parent []string
	parent = append(parent, s.parent...)

	return ScopedName{
		name:         s.name,
		parent:       parent,
		fullPath:     s.fullPath,
		platformName: platforms,
	}
}

func (s ScopedName) String() string {
	return s.fullPath
}

func (s ScopedName) ProtoName() string {
	return s.name
}

func (s ScopedName) CanResolveParent() bool {
	return s.fullPath != ""
}

func (s ScopedName) LocalPath() []string {
	return s.localPath
}
