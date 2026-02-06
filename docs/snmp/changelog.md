# Changelog

Toutes les modifications notables de ce projet sont documentées dans ce fichier.

Le format est basé sur [Keep a Changelog](https://keepachangelog.com/fr/1.0.0/),
et ce projet adhère au [Semantic Versioning](https://semver.org/lang/fr/).

## [1.0.0] - 2026-02-02

### Ajouté

#### Client SNMP
- Support complet des versions SNMPv1, SNMPv2c et SNMPv3
- Opérations GET, GET-NEXT, GET-BULK, SET
- Fonction Walk avec support automatique GET-BULK pour v2c/v3
- Authentification USM pour SNMPv3 (MD5, SHA)
- Chiffrement pour SNMPv3 (DES, AES-128)
- Niveaux de sécurité: NoAuthNoPriv, AuthNoPriv, AuthPriv

#### Encodage
- Implémentation complète du protocole BER (Basic Encoding Rules)
- Support de tous les types ASN.1/SNMP:
  - INTEGER, OCTET STRING, NULL, OBJECT IDENTIFIER
  - IP Address, Counter32, Gauge32, TimeTicks, Counter64
  - NoSuchObject, NoSuchInstance, EndOfMibView
- Encodage et décodage des PDUs SNMP
- Parser et manipulation d'OIDs

#### Pool de connexions
- Gestion de pool de connexions réutilisables
- Health checks périodiques configurables
- Éviction des connexions inactives
- Statistiques et métriques du pool

#### Trap Listener
- Réception de traps SNMPv1, SNMPv2c
- Filtrage par community string
- Support des traps génériques et enterprise-specific
- Handler asynchrone pour traitement non-bloquant

#### Métriques
- Compteurs: requêtes totales, succès, erreurs, timeouts
- Compteurs par opération: GET, GET-NEXT, GET-BULK, SET, WALK
- Jauges: connexions actives, taille du pool
- Histogrammes de latence avec percentiles

#### CLI (edgeo-snmp)
- Commande `info`: informations système de base
- Commande `get`: requête GET simple ou multiple
- Commande `getnext`: requête GET-NEXT
- Commande `getbulk`: requête GET-BULK (v2c/v3)
- Commande `walk`: parcours de sous-arbre MIB
- Commande `bulkwalk`: walk optimisé avec GET-BULK
- Commande `set`: modification de valeurs
- Commande `trap-listen`: écoute des traps
- Commande `version`: affichage de la version
- Support des formats de sortie: texte et JSON
- Mode verbose pour le debugging
- Support des couleurs dans le terminal

#### Documentation
- Documentation complète en français
- Guide de démarrage rapide
- Documentation API détaillée
- Exemples de code

### Infrastructure

- Build system avec Makefile
- Support cross-compilation (Linux, macOS, Windows)
- Architectures supportées: amd64, arm64

## [Unreleased]

### Prévu
- Support SNMPv3 Inform
- Support SNMP over TLS (RFC 6353)
- MIB parser pour résolution de noms
- Support IPv6
- Intégration OpenTelemetry native
- Tests d'intégration avec simulateur SNMP

---

## Convention de versioning

Ce projet suit le [Semantic Versioning](https://semver.org/):

- **MAJOR** : changements incompatibles avec les versions précédentes
- **MINOR** : nouvelles fonctionnalités rétrocompatibles
- **PATCH** : corrections de bugs rétrocompatibles

## Liens

- [Documentation](index.md)
- [Démarrage rapide](getting-started.md)
- [GitHub Repository](https://github.com/edgeo-scada/snmp)
