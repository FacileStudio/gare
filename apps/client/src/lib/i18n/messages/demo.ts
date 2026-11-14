import type { Row } from '../row';

/** The labels inside the landing's demos; commands, commits and log lines stay as gare prints them. */
export const demo = {
	'demo.label': [
		'Gare deploys an app: a click on Deploy streams the build and health check log and the new commit runs. A later push fails its health check and the previous build comes back.',
		"Gare déploie une app : un clic sur Déployer affiche le log de build et de santé, puis le nouveau commit tourne. Un push suivant rate sa vérification de santé et la version d'avant revient.",
		'Gare despliega una app: un clic en Desplegar muestra el log de build y de salud, y el nuevo commit queda en marcha. Un push posterior falla la comprobación de salud y vuelve la versión anterior.',
		'Gare deployt eine App: Ein Klick auf Deployen zeigt das Build- und Health-Check-Log, dann läuft der neue Commit. Ein späterer Push besteht den Health Check nicht und die vorige Version kehrt zurück.'
	],
	'demo.apps': ['Apps', 'Apps', 'Apps', 'Apps'],
	'demo.newApp': ['New app', 'Nouvelle app', 'Nueva app', 'Neue App'],
	'demo.type': ['Type', 'Type', 'Tipo', 'Typ'],
	'demo.commit': ['Commit', 'Commit', 'Commit', 'Commit'],
	'demo.deploy': ['Deploy', 'Déployer', 'Desplegar', 'Deployen'],
	'demo.deploying': ['Starting deploy', 'Lancement', 'Iniciando', 'Startet'],
	'demo.running': ['Running', 'Active', 'En marcha', 'Läuft'],
	'demo.inProgress': ['Running', 'En cours', 'En curso', 'Läuft'],
	'demo.starting': ['Starting', 'Démarrage', 'Arrancando', 'Startet'],
	'demo.log': ['Deploy log', 'Log de déploiement', 'Log de despliegue', 'Deploy-Log'],
	'demo.succeeded': ['Succeeded', 'Réussi', 'Correcto', 'Erfolgreich'],
	'demo.rolledBack': ['Rolled back', 'Annulé', 'Revertido', 'Zurückgesetzt'],
	'demo.live': ['Live', 'En ligne', 'En línea', 'Live'],
	'demo.justNow': ['just now by webhook', "à l'instant par webhook", 'ahora mismo por webhook', 'gerade eben per Webhook'],
	'demo.yesterday': ['yesterday by webhook', 'hier par webhook', 'ayer por webhook', 'gestern per Webhook'],
	'demo.hoursAgo': ['2 hours ago by webhook', 'il y a 2 heures par webhook', 'hace 2 horas por webhook', 'vor 2 Stunden per Webhook'],
	'demo.checking': ['Checking health...', 'Test de santé...', 'Comprobando salud...', 'Prüft Health...'],
	'demo.failed': ['Health check failed on /', 'Échec du test de santé sur /', 'Comprobación fallida en /', 'Health Check auf / fehlgeschlagen'],
	'demo.add': ['Add', 'Ajouter', 'Añadir', 'Hinzufügen'],
	'demo.adding': ['Adding', 'Ajout', 'Añadiendo', 'Hinzufügen...'],
	'demo.push.title': ['Deploys on git push', 'Déploie à chaque git push', 'Despliega con cada git push', 'Deployt bei jedem git push'],
	'demo.push.body': [
		'A signed GitHub or GitLab webhook starts the deploy.',
		'Un webhook GitHub ou GitLab signé lance le déploiement.',
		'Un webhook firmado de GitHub o GitLab lanza el despliegue.',
		'Ein signierter GitHub- oder GitLab-Webhook startet das Deployment.'
	],
	'demo.rollback.title': ['Undoes a bad deploy', 'Annule un mauvais déploiement', 'Deshace un mal despliegue', 'Macht ein schlechtes Deployment rückgängig'],
	'demo.rollback.body': [
		'When a health check fails, the last good build runs again.',
		'Si la vérification de santé échoue, la dernière bonne version repart.',
		'Si la comprobación de salud falla, vuelve la última versión buena.',
		'Schlägt der Health Check fehl, läuft wieder der letzte gute Build.'
	],
	'demo.domain.title': ['A certificate per domain', 'Un certificat par domaine', 'Un certificado por dominio', 'Ein Zertifikat pro Domain'],
	'demo.domain.body': [
		'Add a hostname and Caddy routes it over HTTPS.',
		'Ajoutez un nom de domaine, Caddy le sert en HTTPS.',
		'Añade un dominio y Caddy lo sirve por HTTPS.',
		'Hostnamen hinzufügen, Caddy liefert ihn über HTTPS aus.'
	]
} satisfies Record<string, Row>;
