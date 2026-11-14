import type { Row } from '../row';

/** The landing's sections below the hero demo. */
export const sections = {
	'landing.tiles.title': [
		'Everything after git push',
		'Tout ce qui suit un git push',
		'Todo lo que pasa tras un git push',
		'Alles nach dem git push'
	],
	'landing.features.title': [
		'Plain tools on your server',
		'Des outils simples sur votre serveur',
		'Herramientas sencillas en tu servidor',
		'Einfache Werkzeuge auf Ihrem Server'
	],
	'landing.features.line': [
		'Podman, systemd and Caddy do the work. Gare wires them together.',
		'Podman, systemd et Caddy font le travail. Gare les relie.',
		'Podman, systemd y Caddy hacen el trabajo. Gare los conecta.',
		'Podman, systemd und Caddy erledigen die Arbeit. Gare verbindet sie.'
	],
	'landing.feature.kinds.title': ['Three kinds of app', "Trois types d'app", 'Tres tipos de app', 'Drei Arten von Apps'],
	'landing.feature.kinds.text': [
		'A Containerfile, a static site or a compose file, straight from the repository.',
		'Un Containerfile, un site statique ou un fichier compose, tirés du dépôt.',
		'Un Containerfile, un sitio estático o un archivo compose, directo del repositorio.',
		'Ein Containerfile, eine statische Seite oder eine Compose-Datei, direkt aus dem Repository.'
	],
	'landing.feature.history.title': ['Every deploy on record', 'Chaque déploiement gardé', 'Cada despliegue registrado', 'Jedes Deployment festgehalten'],
	'landing.feature.history.text': [
		'The commit, the trigger and the full log of the last 50 deploys.',
		'Le commit, le déclencheur et le log complet des 50 derniers déploiements.',
		'El commit, el origen y el log completo de los últimos 50 despliegues.',
		'Commit, Auslöser und vollständiges Log der letzten 50 Deployments.'
	],
	'landing.feature.env.title': ['Env vars in one file', 'Les variables dans un fichier', 'Variables en un archivo', 'Variablen in einer Datei'],
	'landing.feature.env.text': [
		'One private file per app, edited from the console or the command line.',
		"Un fichier privé par app, modifié depuis la console ou la ligne de commande.",
		'Un archivo privado por app, editado desde la consola o la línea de comandos.',
		'Eine private Datei pro App, bearbeitet in der Konsole oder auf der Kommandozeile.'
	],
	'landing.feature.rootless.title': ['No Docker daemon', 'Sans démon Docker', 'Sin demonio de Docker', 'Ohne Docker-Daemon'],
	'landing.feature.rootless.text': [
		'Each app is a systemd user unit running rootless Podman.',
		'Chaque app est une unité systemd utilisateur qui lance Podman sans root.',
		'Cada app es una unidad systemd de usuario que ejecuta Podman sin root.',
		'Jede App ist eine systemd-Benutzereinheit mit rootless Podman.'
	],
	'landing.with.antenne': [
		'A deploy fails, Antenne tells you.',
		'Un déploiement échoue, Antenne vous prévient.',
		'Un despliegue falla y Antenne te avisa.',
		'Ein Deployment schlägt fehl, Antenne meldet es.'
	],
	'landing.install.title': ['Install Gare', 'Installer Gare', 'Instalar Gare', 'Gare installieren'],
	'landing.install.text': [
		'One binary for Linux. It needs Podman 5, systemd, Caddy and git.',
		'Un seul binaire pour Linux. Il demande Podman 5, systemd, Caddy et git.',
		'Un solo binario para Linux. Necesita Podman 5, systemd, Caddy y git.',
		'Eine einzige Binärdatei für Linux. Sie braucht Podman 5, systemd, Caddy und git.'
	],
	'landing.plan.title': ['What it costs', 'Combien ça coûte', 'Cuánto cuesta', 'Was es kostet'],
	'landing.plan.note': [
		'Free and open source (GPL-3.0). You install it on your own Linux server.',
		"Gratuit et open source (GPL-3.0). Vous l'installez sur votre propre serveur Linux.",
		'Gratis y de código abierto (GPL-3.0). Lo instalas en tu propio servidor Linux.',
		'Kostenlos und Open Source (GPL-3.0). Sie installieren es auf Ihrem eigenen Linux-Server.'
	],
	'landing.plan.suite': [
		'Need mail, files, quotes or a calendar too? The Facile suite has 25 work apps on one account, with nothing to install.',
		"Besoin aussi de mails, de fichiers, de devis ou d'un agenda ? La suite Facile réunit 25 apps sur un seul compte, sans rien installer.",
		'¿También necesitas correo, archivos, presupuestos o un calendario? La suite Facile reúne 25 apps en una sola cuenta, sin instalar nada.',
		'Brauchen Sie auch Mail, Dateien, Angebote oder einen Kalender? Die Facile-Suite bündelt 25 Apps in einem Konto, ohne Installation.'
	],
	'landing.plan.suiteLink': ['See the Facile suite', 'Découvrir la suite Facile', 'Ver la suite Facile', 'Facile-Suite ansehen'],
	'landing.dev.description': [
		"Podman with no Docker daemon, systemd, and automatic HTTPS with Caddy. The web console is for your server's admins.",
		'Podman sans démon Docker, systemd, et HTTPS automatique avec Caddy. La console web sert aux admins de votre serveur.',
		'Podman sin demonio de Docker, systemd y HTTPS automático con Caddy. La consola web es para los admins de tu servidor.',
		'Podman ohne Docker-Daemon, systemd und automatisches HTTPS mit Caddy. Die Webkonsole ist für die Admins Ihres Servers.'
	],
	'landing.dev.cli': [
		'The gare command does what the console does: create, deploy, env vars, domains, logs.',
		'La commande gare fait tout ce que fait la console : créer, déployer, variables, domaines, logs.',
		'El comando gare hace lo mismo que la consola: crear, desplegar, variables, dominios, logs.',
		'Der Befehl gare kann alles, was die Konsole kann: anlegen, deployen, Variablen, Domains, Logs.'
	],
	'landing.dev.source': [
		'The whole code is on GitHub under GPL-3.0.',
		'Tout le code est sur GitHub, sous GPL-3.0.',
		'Todo el código está en GitHub bajo GPL-3.0.',
		'Der gesamte Code liegt unter GPL-3.0 auf GitHub.'
	]
} satisfies Record<string, Row>;
