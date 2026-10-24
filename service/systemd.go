package service

import kservice "github.com/kardianos/service"

// Config extends the servicego default service config with a custom systemd
// unit template. servicego rewrites the unit on every deploy, so this is the
// place to change it.
func (sv *Service) Config() *kservice.Config {
	cfg := sv.Defaults.Config()
	cfg.Option["SystemdScript"] = systemdScript
	return cfg
}

// systemdScript is kardianos/service's default systemd unit template (v1.2.4)
// plus KillMode=mixed.
//
// With the default KillMode=control-group, "systemctl stop" sends SIGTERM to
// every process in the unit's cgroup at once: factop, the Factorio server and
// every plugin. Plugins then exit on their own, racing the PluginManager's
// ordered shutdown, which logs them as "exited unexpectedly" and schedules a
// restart. KillMode=mixed sends SIGTERM to factop only, so factop stops the
// plugins (reverse order) and the Factorio server itself; anything still left
// in the cgroup once factop exits is sent SIGKILL.
const systemdScript = `[Unit]
Description={{.Description}}
ConditionFileIsExecutable={{.Path|cmdEscape}}
{{range $i, $dep := .Dependencies}} 
{{$dep}} {{end}}

[Service]
StartLimitInterval=5
StartLimitBurst=10
ExecStart={{.Path|cmdEscape}}{{range .Arguments}} {{.|cmd}}{{end}}
{{if .ChRoot}}RootDirectory={{.ChRoot|cmd}}{{end}}
{{if .WorkingDirectory}}WorkingDirectory={{.WorkingDirectory|cmdEscape}}{{end}}
{{if .UserName}}User={{.UserName}}{{end}}
{{if .ReloadSignal}}ExecReload=/bin/kill -{{.ReloadSignal}} "$MAINPID"{{end}}
{{if .PIDFile}}PIDFile={{.PIDFile|cmd}}{{end}}
{{if and .LogOutput .HasOutputFileSupport -}}
StandardOutput=file:{{.LogDirectory}}/{{.Name}}.out
StandardError=file:{{.LogDirectory}}/{{.Name}}.err
{{- end}}
{{if gt .LimitNOFILE -1 }}LimitNOFILE={{.LimitNOFILE}}{{end}}
{{if .Restart}}Restart={{.Restart}}{{end}}
{{if .SuccessExitStatus}}SuccessExitStatus={{.SuccessExitStatus}}{{end}}
RestartSec=120
KillMode=mixed
EnvironmentFile=-/etc/sysconfig/{{.Name}}

{{range $k, $v := .EnvVars -}}
Environment={{$k}}={{$v}}
{{end -}}

[Install]
WantedBy=multi-user.target
`
