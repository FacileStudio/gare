import type { CliCommand, DownloadEntry } from '@facile/muse';
import { version } from '../../package.json';

const RELEASE = `https://github.com/FacileStudio/gare/releases/download/v${version}`;

/** Every release, older versions included. */
export const RELEASES_URL = 'https://github.com/FacileStudio/gare/releases';

/** The SHA-256 sums GoReleaser publishes beside the archives of this client's release. */
export const CHECKSUMS_URL = `${RELEASE}/checksums.txt`;

/** The Linux archives of the release this client ships in: the binary plus the built web client. */
export const DOWNLOADS: DownloadEntry[] = (['amd64', 'arm64'] as const).map((arch) => ({
	platform: 'linux',
	href: `${RELEASE}/gare_${version}_linux_${arch}.tar.gz`,
	label: 'tar.gz',
	arch: arch === 'amd64' ? 'x86_64' : 'ARM64',
	version
}));

/** The two install paths the README documents, both ending in `facile install gare`. */
export const INSTALL_COMMANDS: CliCommand[] = [
	{ label: 'Install script', command: 'curl -fsSL https://raw.githubusercontent.com/FacileStudio/gare/main/install.sh | bash' },
	{ label: 'With facile', command: 'facile install gare' }
];
