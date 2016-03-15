import os

"""DEFUNCT
darwin    arm
darwin    arm64
dragonfly    amd64
freebsd    386
freebsd    amd64
freebsd    arm
linux    386
linux    arm64
linux    ppc64le
netbsd    386
netbsd    amd64
netbsd    arm
openbsd    386
openbsd    amd64
openbsd    arm
plan9    386
plan9    amd64
solaris    amd64
windows    386
darwin    386
darwin    amd64
linux    arm
linux    ppc64
windows    amd64"""

arches = """linux    amd64
windows amd64
linux    arm
darwin    amd64"""

arches = arches.split("\n")
version = "1.0"
programName = "awwkoala"
try:
    os.system("rm -rf builds")
except:
    pass
os.mkdir("builds")

for arch in arches:
    goos = arch.split()[0]
    goarch = arch.split()[1]
    exe = ""
    if "windows" in goos:
        exe = ".exe"
    cmd1  = 'env GOOS=%(goos)s GOARCH=%(goarch)s go build -o builds/%(programName)s%(exe)s' % {'goos':goos,'goarch':goarch,'exe':exe,'programName':programName}
    cmd2 = 'zip -r %(programName)s-%(version)s-%(goos)s-%(goarch)s.zip %(programName)s%(exe)s ../templates ../static' % {'goos':goos,'goarch':goarch,'exe':exe,'version':version,'programName':programName}
    print(cmd1)
    os.system(cmd1)
    os.chdir("builds")
    print(cmd2)
    os.system(cmd2)
    cmd3 = 'rm %(programName)s%(exe)s' % {'exe':exe,'programName':programName}
    print(cmd3)
    os.system(cmd3)
    os.chdir("../")
