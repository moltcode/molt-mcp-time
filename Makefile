NAME    := molt-mcp-time
VERSION := $(shell python3 -c "import json;print(json.load(open('package.json'))['version'])")

PLATFORMS := darwin-arm64 darwin-x64 linux-x64 linux-arm64 win32-x64

# npm platform id -> GOOS/GOARCH
goos   = $(word 1,$(subst -, ,$(1)))
goarch = $(word 2,$(subst -, ,$(1)))
GOOS_darwin := darwin
GOOS_linux  := linux
GOOS_win32  := windows
GOARCH_arm64 := arm64
GOARCH_x64   := amd64

.PHONY: build dist clean

build:
	@for p in $(PLATFORMS); do \
		os=$${p%%-*}; arch=$${p##*-}; \
		case $$os in win32) goos=windows;; *) goos=$$os;; esac; \
		case $$arch in x64) goarch=amd64;; *) goarch=$$arch;; esac; \
		ext=""; [ "$$goos" = "windows" ] && ext=".exe"; \
		echo "building dist/$$p/$(NAME)$$ext"; \
		mkdir -p dist/$$p; \
		CGO_ENABLED=0 GOOS=$$goos GOARCH=$$goarch go build -trimpath -ldflags "-s -w" -o dist/$$p/$(NAME)$$ext .; \
	done

# npm-layout tarball (package/ root) so the same artifact can later be
# published to the npm registry unchanged.
dist: build
	rm -rf build $(NAME)-$(VERSION).tgz
	mkdir -p build/package
	cp package.json README.md build/package/
	cp -R dist build/package/dist
	tar -czf $(NAME)-$(VERSION).tgz -C build package
	shasum -a 256 $(NAME)-$(VERSION).tgz

clean:
	rm -rf dist build $(NAME)-*.tgz
