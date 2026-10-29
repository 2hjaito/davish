// ufw commands for the Ports app, typed into a terminal for the user to run.
import type { Port } from '../../lib/api/ports'

export const openCommand = (port: number) => `sudo ufw allow ${port}/tcp`

/** Removes the allow rule, written with or without /tcp. */
export const closeCommand = (port: number) =>
  `sudo ufw delete allow ${port}/tcp ; sudo ufw delete allow ${port}`

/** Ports of admin tools and databases that should not face the Internet. */
export const ADMIN_PORTS: Record<number, string> = {
  2375: 'Docker API',
  3000: 'Grafana',
  3306: 'MySQL / MariaDB',
  5432: 'PostgreSQL',
  5601: 'Kibana',
  6379: 'Redis',
  8086: 'InfluxDB',
  9000: 'Portainer / MinIO',
  9090: 'Prometheus',
  9200: 'Elasticsearch',
  11211: 'Memcached',
  15672: 'RabbitMQ',
  27017: 'MongoDB',
}

/** Exposed admin ports: those ufw can close, and containers' ones, which
 *  ufw cannot (docker's rules come first) and need rebinding to 127.0.0.1. */
export function exposedAdmin(ports: readonly Port[]): { ufw: Port[]; docker: Port[] } {
  const hit = ports.filter((p) => p.exposed === 'yes' && ADMIN_PORTS[p.port])
  const seen = new Set<number>()
  const once = (p: Port) => !seen.has(p.port) && seen.add(p.port)
  return {
    ufw: hit.filter((p) => !p.container && p.process !== 'docker-proxy').filter(once),
    docker: hit.filter((p) => p.container || p.process === 'docker-proxy'),
  }
}

/** One line closing every port in ports. */
export const closeAllCommand = (ports: readonly Port[]) =>
  ports.map((p) => closeCommand(p.port)).join(' ; ')
