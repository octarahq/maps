# Octara Maps OSM+PRIM Provider

Outil Go isolé (`maps/provider/`) qui construit et maintient notre propre provider de données cartographiques pour l'Île-de-France en croisant **OpenStreetMap**, **PRIM / IDFM**, **SNCF Open Data**, **IGN BD TOPO 3D** et l'analyse agentique (Antigravity) pour les cas ambigus, les guichets, les marqueurs de quai et la géométrie indoor/3D.

## Architecture & Flux

1. **`fetch`** (`make provider-fetch`)
   - Télécharge les jeux de données IDFM/GTFS (`arrets`, `arrets-lignes`, `zones-de-correspondance`, `zones-d-arrets`, `acces`, `relations-acces`, `points-de-vente`, `emplacement-des-gares-idf`, `accessibilite-en-gare`, `positionnement-dans-la-rame`, `stationnement-velo-en-ile-de-france`, `referentiel-des-lignes`, `fontaines`, `projets_arrets_idf`) et SNCF (`liste-des-quais` GeoJSON + `SNCF_GC_accessibilite_gares.xml` NeTEx).
   - Réutilise `../data/osm/ile-de-france-latest.osm.pbf` si présent (ou télécharge Geofabrik).
   - Convertit tout plan PDF/SVG déposé dans `data/plans/` en images PNG dans `data/plans/rendered/`.

2. **`sync`** (`make provider-sync PROVIDER_ARGS='--station "Châtelet"'`)
   - Exécute le **merge 3-voies** entre le snapshot baseline (`data/state/state.db`), le nouveau `.osm.pbf` et `data/overlays/overrides.json` (`osm_id` + ancre sémantique).
   - Croise OSM ↔ IDFM ↔ SNCF ↔ IGN 3D.
   - Écrit les patchs haute confiance dans `data/staging/auto_*.json`.
   - Exporte tous les cas ambigus, conflits, manques indoor/3D/guichets/marqueurs de quai et plans de gares dans `data/pending_cases/` (`cases_index.json`, `stations/<slug>.json`, `stations/<slug>_context.geojson`).

3. **Analyse agentique (Antigravity)**
   - Antigravity lit `data/pending_cases/cases_index.json`, les bundles `stations/<slug>.json`, les contextes GeoJSON et les images de plans dans `data/plans/rendered/`.
   - Il rédige des propositions dans `data/staging/agent_<slug>.json` au format `StagingBatch` (`OverlayPatch` avec tags OSM Simple Indoor Tagging `level=*`, `height=*`, `indoor=*`, `shop=ticket`, `local_ref`, etc.).

4. **`review` / `promote`** (`make provider-review` ou `make provider-review-cli`)
   - UI Web locale (`http://127.0.0.1:8091`) avec carte interactive, sélecteur d'étage `level`, diff tags avant/après et édition directe.
   - Mode CLI (`--cli`) pour valider rapidement en batch.
   - Promouvoir vers `data/overlays/overrides.json`.

5. **`build`** (`make provider-build`)
   - Alloue des IDs synthétiques positifs persistants (`>= 10_000_000_000_001`) compatibles Tilemaker/Planetiler/Osmium/`paulmach/osm`.
   - Génère `data/out/osm_export.osc`, compile `data/out/ile-de-france-enriched.osm.pbf` via `osmium apply-changes`, et produit `data/out/stations/<slug>.geojson` + `stations_index.json`.

6. **`serve`** (`make provider-serve`)
   - Démarre l'API HTTP (`http://127.0.0.1:8092`) exposant `/stations`, `/stations/:id/levels`, `/stations/:id/geojson?level=...`, `/pbf/latest`, `/diff/osc`.
