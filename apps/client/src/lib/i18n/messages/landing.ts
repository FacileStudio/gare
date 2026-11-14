import type { Row } from '../row';

/** The public landing's head, hero, header and footer. */
export const landing = {
	'landing.title': [
		'Gare · Deploy from git to your own server',
		'Gare · Déployez depuis git, sur votre serveur',
		'Gare · Despliega desde git, en tu propio servidor',
		'Gare · Aus git deployen, auf Ihren eigenen Server'
	],
	'landing.description': [
		'Push to git: Gare builds the app, puts it live and rolls back to the last version if it breaks.',
		"Poussez sur git : Gare construit l'app, la met en ligne et revient à la version d'avant si elle plante.",
		'Haz push en git: Gare construye la app, la publica y vuelve a la versión anterior si falla.',
		'Push auf git: Gare baut die App, bringt sie online und kehrt zur letzten Version zurück, wenn sie abstürzt.'
	],
	'landing.nav.features': ['Features', 'Fonctions', 'Funciones', 'Funktionen'],
	'landing.nav.install': ['Install', 'Installer', 'Instalar', 'Installieren'],
	'landing.hero.line1': ['Deploy from git', 'Déployez depuis git,', 'Despliega desde git,', 'Aus git deployen,'],
	'landing.hero.line2': [
		'to your own server.',
		'sur votre serveur.',
		'en tu propio servidor.',
		'auf Ihren eigenen Server.'
	],
	'landing.hero.install': ['Install Gare', 'Installer Gare', 'Instalar Gare', 'Gare installieren'],
	'landing.hero.console': [
		'Already installed? Open your console',
		'Déjà installé ? Ouvrez votre console',
		'¿Ya lo tienes instalado? Abre tu consola',
		'Schon installiert? Öffnen Sie Ihre Konsole'
	],
	'landing.header.console': ['Open your console', 'Ouvrir la console', 'Abrir la consola', 'Konsole öffnen'],
	'landing.footer.tagline': [
		'Deploy your apps from git to your own server.',
		'Déployez vos apps depuis git, sur votre serveur.',
		'Despliega tus apps desde git, en tu servidor.',
		'Deployen Sie Ihre Apps aus git auf Ihren Server.'
	],
	'landing.footer.downloads': ['Downloads', 'Téléchargements', 'Descargas', 'Downloads'],
	'landing.footer.source': ['Source code', 'Code source', 'Código fuente', 'Quellcode'],
	'landing.footer.suite': ['Facile suite', 'Suite Facile', 'Suite Facile', 'Facile-Suite'],
	'landing.cta.title': [
		'Put your next push live',
		'Mettez votre prochain push en ligne',
		'Publica tu próximo push',
		'Bringen Sie Ihren nächsten Push online'
	],
	'landing.cta.text': [
		"Install Gare on your server. Your server's admins then sign in to the console to see every app, its deploys and its logs.",
		'Installez Gare sur votre serveur. Ses admins se connectent ensuite à la console pour voir chaque app, ses déploiements et ses logs.',
		'Instala Gare en tu servidor. Sus admins entran luego en la consola para ver cada app, sus despliegues y sus logs.',
		'Installieren Sie Gare auf Ihrem Server. Dessen Admins melden sich dann in der Konsole an und sehen jede App, ihre Deployments und Logs.'
	],
	'install.script': ['Install script', "Script d'installation", 'Script de instalación', 'Installationsskript'],
	'install.facile': ['With facile', 'Avec facile', 'Con facile', 'Mit facile'],
	'downloads.title': ['Downloads · Gare', 'Téléchargements · Gare', 'Descargas · Gare', 'Downloads · Gare'],
	'downloads.heading': ['Downloads', 'Téléchargements', 'Descargas', 'Downloads']
} satisfies Record<string, Row>;
