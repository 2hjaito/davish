import type { Dict } from "./en";

export const fr: Dict = {
  lang: "fr",
  title: "Davish : un bureau pour vos serveurs Linux",
  description:
    "Terminal, fichiers, Docker, journaux, bases de données et bien plus, via un simple SSH, dans une seule app de bureau pour macOS, Windows et Linux. Rien à installer sur le serveur.",
  nav: {
    features: "Fonctionnalités",
    security: "Sécurité",
    download: "Télécharger",
    docs: "Documentation",
    theme: "Clair ou sombre",
    changelog: "Nouveautés",
  },
  hero: {
    title: "Un bureau pour vos serveurs Linux.",
    lead: "Terminal, fichiers, services, Docker, journaux, bases de données et git via un simple SSH, réunis dans un seul endroit qui se comporte comme un système d’exploitation.",
    tags: ["Aucun panneau web", "Aucune connexion", "Aucun agent"],
    download: "Télécharger pour",
    other: "Autres plateformes",
    github: "Voir sur GitHub",
    version: (v: string) => `Version ${v} · Gratuit et open source (GPL-3.0)`,
    shotAlt:
      "Le bureau Davish : Terminal, Files et Monitor sur un même serveur",
    badge: (v: string) => `Nouveautés de la version ${v}`,
    docs: "Lire la documentation",
    platforms: "Fonctionne sur macOS, Windows et Linux",
    hint: "Choisissez une app pour jeter un œil à l’intérieur",
    tabs: {
      desktop: "Bureau",
      servers: "Tous les serveurs",
      docker: "Docker",
      database: "Bases de données",
      git: "Git",
      security: "Sécurité",
    },
  },
  why: {
    title: "Pourquoi Davish",
    items: [
      {
        title: "On ouvre, et on y est",
        text: "Pas d’écran de connexion, pas de panneau web. Davish se connecte avec votre clé SSH et vous place directement sur le bureau du serveur. Rien de nouveau n’écoute sur le serveur.",
      },
      {
        title: "Sans agent",
        text: "Seuls sshd et SFTP sont nécessaires. Tout s’exécute depuis votre machine ; rien n’est copié, installé ni laissé sur le serveur.",
      },
      {
        title: "Chaque commande est visible",
        text: "Toute action destructive affiche d’abord la commande exacte. Le journal des commandes garde tout et exporte les modifications sous forme de script.",
      },
      {
        title: "Plusieurs serveurs, un seul bureau",
        text: "Chaque serveur est un espace de travail qui se souvient de ses fenêtres. Glissez des fichiers d’un serveur à l’autre.",
      },
      {
        title: "Votre config ssh fonctionne telle quelle",
        text: "Importez vos hôtes depuis ~/.ssh/config ou un inventaire Ansible, clés et ProxyJump compris.",
      },
      {
        title: "Se met à jour tout seul",
        text: "Une nouvelle version indique ce qui a changé ; Davish la télécharge, vérifie sa signature et redémarre dessus.",
      },
    ],
  },
  features: {
    title:
      "Tout ce qu’il faut à un serveur, dans des fenêtres que vous connaissez déjà",
    lead: "Plus de vingt apps partagent un même bureau : déplacez-les, ancrez-les et passez de l’une à l’autre comme sur votre propre ordinateur.",
    show: [
      {
        key: "servers",
        title: "Tous vos serveurs d’un coup d’œil",
        text: "L’état de santé et les alertes de chaque serveur dans une seule fenêtre, avec une carte de ce que chacun héberge : domaine, nginx, port, conteneur.",
      },
      {
        key: "multi-run",
        title: "Une commande, plusieurs serveurs",
        text: "Lancez une commande sur plusieurs serveurs à la fois, chacun dans son onglet de terminal, avec un bouton d’arrêt qui arrête vraiment.",
      },
      {
        key: "git",
        title: "Git sur le serveur",
        text: "État, diffs et graphe des commits des dépôts de votre serveur, avec fetch et pull.",
      },
      {
        key: "docker",
        title: "Docker, en toute clarté",
        text: "Les conteneurs par projet compose, les images et leurs couches, les volumes, les journaux et un shell en un clic.",
      },
      {
        key: "sqlite-rows",
        title: "Des bases de données sans panneau web",
        text: "MariaDB, PostgreSQL, MongoDB et SQLite : création, dump et restauration, et lecture des tables SQLite page par page.",
      },
      {
        key: "security",
        title: "Qui frappe à la porte",
        text: "Les échecs de connexion par heure et par adresse, les sessions ouvertes, ufw et fail2ban, sans quitter le bureau.",
      },
    ],
    more: "Et aussi",
    list: [
      "Terminal",
      "Fichiers (SFTP)",
      "Éditeur",
      "Journaux",
      "Moniteur",
      "Services",
      "Cron",
      "Nginx",
      "Paquets",
      "Ports et tunnels",
      "Navigateur",
      "Disque",
      "Clés SSH",
      "Commandes enregistrées",
      "Journal des commandes",
      "Spotlight et Launchpad",
    ],
    all: "Voir toutes les fonctionnalités",
  },
  security: {
    title: "Conçu pour mériter la confiance du root",
    items: [
      "Mots de passe et phrases secrètes ne vivent que dans le trousseau de votre système.",
      "Le mot de passe sudo n’est jamais enregistré : demandé une fois par session, gardé en mémoire, oublié au bout de 15 minutes.",
      "Se verrouille avec son propre mot de passe ou Touch ID quand vous vous éloignez.",
      "Les mises à jour sont vérifiées : le programme de mise à jour contrôle la signature de la version avant d’installer quoi que ce soit.",
    ],
    more: "Lire les notes de sécurité",
  },
  download: {
    title: "Télécharger Davish",
    lead: "Gratuit pour macOS, Windows et Linux. Votre serveur n’a besoin que d’OpenSSH avec SFTP.",
    platforms: {
      macos: {
        name: "macOS",
        note: "macOS 12 ou ultérieur, Apple silicon et Intel",
      },
      windows: { name: "Windows", note: "Windows 10 ou ultérieur, 64 bits" },
      linux: { name: "Linux", note: "GTK 3 et WebKitGTK 4.1, x86-64" },
    },
    installer: "Programme d’installation",
    archive: "Version portable",
    checksums: "Sommes de contrôle",
    notes: "Notes de version",
    unsigned:
      "Davish n’est pas encore signé numériquement ; macOS et Windows affichent donc un avertissement la première fois. Le guide d’installation sur GitHub explique comment l’ouvrir.",
    guide: "Guide d’installation",
  },
  stats: {
    downloads: "téléchargements de l’app",
    apps: "apps sur un même bureau",
    version: (date: string) => `dernière version, ${date}`,
    stars: "étoiles sur GitHub",
    free: "Gratuit",
    freeNote: "open source, GPL-3.0",
  },
  docs: {
    title: "Prenez-le en main en dix minutes",
    lead: "Chaque app a sa propre page dans la documentation : ce qu’elle fait, ce qu’elle exécute sur le serveur et où restent vos données.",
    start: "Pour commencer",
    links: [
      { label: "Installer Davish", href: "start/install/" },
      { label: "Connecter votre premier serveur", href: "start/first-server/" },
      { label: "Ce dont le serveur a besoin", href: "start/requirements/" },
      { label: "Raccourcis clavier", href: "start/shortcuts/" },
    ],
  },
  changelog: {
    title: "Nouveautés · Davish",
    description:
      "Ce qui change à chaque version de Davish, de la plus récente à la plus ancienne.",
    heading: "Nouveautés",
    lead: "Ce qui change à chaque version, de la plus récente à la plus ancienne.",
    englishOnly: "Les nouveautés sont en anglais.",
  },
  voices: {
    title: "Ce qu’en disent les utilisateurs",
    note: "Extraits de publications et d’issues publiques.",
    translated: (from: "en" | "zh") =>
      from === "zh" ? "Traduit du chinois" : "Traduit de l’anglais",
    more: (n: number) =>
      n < 2 ? `Voir ${n} autre avis` : `Voir ${n} autres avis`,
    less: "Afficher moins",
  },
  community: {
    discord: "Rejoignez-nous sur Discord",
    discordNote: "Questions, idées et versions en avant-première.",
  },
  thanks: {
    title: "Fait avec celles et ceux qui l’utilisent",
    lead: "Code, rapports de bugs, tests sur leurs propres machines et idées : merci.",
    code: "code",
    issue: "issue",
    discussion: "discussion",
    join: "Les rejoindre sur GitHub",
  },
  footer: {
    license: "Davish est un logiciel libre sous licence GPL-3.0.",
    changelog: "Nouveautés",
    issues: "Signaler un problème",
    language: "Langue",
  },
};
