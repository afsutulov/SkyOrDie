// The build command cross-compiles SkyOfDie with Go alone. Windows icons are
// generated as temporary COFF resources; macOS targets are packaged as .app.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/tc-hib/winres"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	all := flag.Bool("all", false, "build all six desktop targets")
	osName := flag.String("os", runtime.GOOS, "target OS: windows, linux, darwin")
	arch := flag.String("arch", runtime.GOARCH, "target architecture: amd64, arm64")
	out := flag.String("out", "build", "output directory")
	ver := flag.String("version", "0.7.3", "embedded application version")
	flag.Parse()
	if strings.ContainsAny(*ver, " \t\r\n\"'") {
		return fmt.Errorf("version must be a single token")
	}
	if _, err := os.Stat("main.go"); err != nil {
		return fmt.Errorf("run this command from the SkyOrDie project root")
	}
	targets := [][2]string{{*osName, *arch}}
	if *all {
		targets = nil
		for _, o := range []string{"windows", "linux", "darwin"} {
			for _, a := range []string{"amd64", "arm64"} {
				targets = append(targets, [2]string{o, a})
			}
		}
	}
	for _, t := range targets {
		if err := build(t[0], t[1], *out, *ver); err != nil {
			return err
		}
	}
	return nil
}
func build(goos, arch, out, version string) error {
	if goos != "windows" && goos != "darwin" && goos != "linux" {
		return fmt.Errorf("unsupported OS %q", goos)
	}
	if arch != "amd64" && arch != "arm64" {
		return fmt.Errorf("unsupported architecture %q", arch)
	}
	dir := filepath.Join(out, goos+"-"+arch)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	output := filepath.Join(dir, "SkyOrDie")
	ld := "-s -w -X main.version=" + version
	if goos == "windows" {
		output += ".exe"
		ld += " -H=windowsgui"
		name := "skyordie_icon_windows_" + arch + ".syso"
		if _, err := os.Stat(name); err == nil {
			return fmt.Errorf("temporary resource %s already exists; remove it before rebuilding", name)
		}
		ico, err := os.Open("assets/icon/SkyOrDie.ico")
		if err != nil {
			return err
		}
		icon, err := winres.LoadICO(ico)
		ico.Close()
		if err != nil {
			return err
		}
		rs := winres.ResourceSet{}
		if err = rs.SetIcon(winres.ID(1), icon); err != nil {
			return err
		}
		f, err := os.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		defer os.Remove(name)
		err = rs.WriteObject(f, winres.Arch(arch))
		ce := f.Close()
		if err != nil {
			return err
		}
		if ce != nil {
			return ce
		}
	}
	if goos == "darwin" {
		app := filepath.Join(dir, "SkyOrDie.app", "Contents")
		if err := os.MkdirAll(filepath.Join(app, "MacOS"), 0755); err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Join(app, "Resources"), 0755); err != nil {
			return err
		}
		output = filepath.Join(app, "MacOS", "SkyOrDie")
		plist := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
<key>CFBundleExecutable</key><string>SkyOrDie</string>
<key>CFBundleIdentifier</key><string>games.skyordie.desktop</string>
<key>CFBundleName</key><string>SkyOrDie</string>
<key>CFBundleDisplayName</key><string>SkyOrDie</string>
<key>CFBundlePackageType</key><string>APPL</string>
<key>CFBundleIconFile</key><string>SkyOrDie.icns</string>
<key>CFBundleShortVersionString</key><string>%s</string>
<key>CFBundleVersion</key><string>%s</string>
<key>LSMinimumSystemVersion</key><string>12.0</string>
<key>NSHighResolutionCapable</key><true/>
</dict></plist>
`, version, version)
		if err := os.WriteFile(filepath.Join(app, "Info.plist"), []byte(plist), 0644); err != nil {
			return err
		}
		if err := copyFile("assets/icon/SkyOrDie.icns", filepath.Join(app, "Resources", "SkyOrDie.icns")); err != nil {
			return err
		}
	}
	env := []string{}
	for _, e := range os.Environ() {
		if !strings.HasPrefix(e, "GOOS=") && !strings.HasPrefix(e, "GOARCH=") && !strings.HasPrefix(e, "CGO_ENABLED=") {
			env = append(env, e)
		}
	}
	cmd := exec.Command("go", "build", "-buildvcs=false", "-trimpath", "-ldflags", ld, "-o", output, ".")
	cmd.Env = append(env, "GOOS="+goos, "GOARCH="+arch, "CGO_ENABLED=0")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	fmt.Printf("Building %s/%s -> %s\n", goos, arch, output)
	if err := cmd.Run(); err != nil {
		return err
	}
	if goos == "linux" {
		if err := copyFile("assets/icon/SkyOrDie.png", filepath.Join(dir, "SkyOrDie.png")); err != nil {
			return err
		}
	}
	return nil
}
func copyFile(src, dst string) error {
	b, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, b, 0644)
}
