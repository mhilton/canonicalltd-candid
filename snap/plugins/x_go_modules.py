import snapcraft


_GOARCH = {"armhf": "arm", "i386": "386", "ppc64el": "ppc64le"}


class XGoModules(snapcraft.BasePlugin):
    @classmethod
    def schema(cls):
        schema = super().schema()
        schema["properties"]["go-packages"] = {
            "type": "array",
            "minitems": 1,
            "uniqueItems": True,
            "items": {"type": "string"},
            "default": [],
        }
        schema["properties"]["go-buildtags"] = {
            "type": "array",
            "minitems": 1,
            "uniqueItems": True,
            "items": {"type": "string"},
            "default": [],
        }
        return schema

    @classmethod
    def get_build_properties(cls):
        props = super().get_build_properties()
        props.extend(['go-buildtags', 'go-packages'])
        return props

    def env(self, root):
        env = super().env(root)
        env.append('GOBIN=' + self.installdir + '/bin')
        env.append('GOARCH=' +
                   _GOARCH.get(self.project.deb_arch, self.project.deb_arch))
        if self.project.deb_arch == "armhf":
            env.append('GOARM=7')
        return env

    def build(self):
        super().build()
        cmd = ['go', 'install']
        if self.project.debug:
            cmd.append('-v')
        if self.options.go_buildtags:
            cmd.extend(["-tags", " ".join(self.options.go_buildtags)])
        cmd.extend(self.options.go_packages)
        self.run(cmd)

    def enable_cross_compilation(self):
        pass
