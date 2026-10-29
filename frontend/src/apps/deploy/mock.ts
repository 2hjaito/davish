// Demo plan for the plain browser preview (vite dev), where there is no Go side.
import type { DeployPlan, DeploySpec } from '../../lib/api/deploy'

export function mockPlan(spec: DeploySpec): Promise<DeployPlan> {
  const domain = spec.domains[0] ?? 'app.example.com'
  const steps = [
    ...(spec.source === 'git'
      ? [
          {
            title: 'clone',
            command: `sudo install -d -o "$USER" -- ${spec.dir} && git clone -- ${spec.repoURL} ${spec.dir}`,
          },
        ]
      : []),
    {
      title: 'systemd',
      command: `printf %s '…' | sudo tee -- /etc/systemd/system/${spec.name}.service > /dev/null && sudo systemctl daemon-reload && sudo systemctl enable --now ${spec.name}`,
      file: `[Service]\nWorkingDirectory=${spec.dir}\nExecStart=${spec.unit.execStart}\n`,
    },
    {
      title: 'nginx',
      command: `printf %s '…' | sudo tee -- /etc/nginx/sites-available/${domain} > /dev/null && …`,
    },
    ...(spec.ssl
      ? [{ title: 'https', command: `sudo certbot --nginx -d ${domain} --redirect` }]
      : []),
  ]
  return Promise.resolve({
    steps,
    line: steps.map((s) => `{ ${s.command}; }`).join(' && '),
    created: [
      spec.dir,
      `/etc/systemd/system/${spec.name}.service`,
      `/etc/nginx/sites-available/${domain}`,
    ],
    uninstall: `sudo rm -f -- /etc/nginx/sites-available/${domain} ; sudo systemctl disable --now ${spec.name}`,
  })
}
