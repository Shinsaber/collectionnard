{
  description = "Collectionnard — environnement de développement (Go, Node, PostgreSQL)";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  };

  outputs = { self, nixpkgs }:
    let
      systems = [ "x86_64-linux" "aarch64-linux" "x86_64-darwin" "aarch64-darwin" ];
      forAllSystems = f: nixpkgs.lib.genAttrs systems (system: f nixpkgs.legacyPackages.${system});
    in
    {
      devShells = forAllSystems (pkgs:
      let

        # PostgreSQL local (alternative à Docker) : données dans ./.pgdata
        pgStart = pkgs.writeShellScriptBin "pg-start" ''
          set -euo pipefail
          export PGDATA="$PWD/.pgdata"
          if [ ! -d "$PGDATA" ]; then
            initdb -U colec --auth=trust -D "$PGDATA" >/dev/null
            pg_ctl -D "$PGDATA" -o "-k $PGDATA -p 5432" -l "$PGDATA/log" start
            createdb -h "$PGDATA" -U colec colec
            (cd server && SEED_ON_EMPTY=true go run . migrate)
            echo "Base initialisée (migrations + seed)."
          else
            pg_ctl -D "$PGDATA" -o "-k $PGDATA -p 5432" -l "$PGDATA/log" start
            (cd server && go run . migrate)   # applique les migrations en attente
          fi
        '';
        pgStop = pkgs.writeShellScriptBin "pg-stop" ''
          pg_ctl -D "$PWD/.pgdata" stop
        '';
        pgReset = pkgs.writeShellScriptBin "pg-reset" ''
          set -euo pipefail
          pg_ctl -D "$PWD/.pgdata" stop 2>/dev/null || true
          rm -rf "$PWD/.pgdata"
          pg-start
        '';
      in
      {
        default = pkgs.mkShell {
          packages = with pkgs; [
            # Backend (Go 1.27 : go.mod exige ≥ 1.26, pgx ≥ 5.9 exige ≥ 1.25)
            go_1_27
            gopls
            gotools
            govulncheck
            golangci-lint
            # Frontend (Node 22 LTS ; Node 20 retiré de nixpkgs)
            nodejs_22
            # Base de données
            postgresql_16
            pgStart
            pgStop
            pgReset
          ];

          DATABASE_URL = "postgres://colec:colec@localhost:5432/colec?sslmode=disable";
          JWT_SECRET = "dev-secret-dev-secret-dev-secret-0000"; # ≥ 32 caractères (exigé par config.go)
          UPLOAD_DIR = "uploads";
          LOG_FORMAT = "text"; # logs lisibles en dev (json par défaut)

          shellHook = ''
            echo "Collectionnard dev shell — go $(go env GOVERSION), node $(node --version)"
            echo "  pg-start | pg-stop | pg-reset   (PostgreSQL local dans ./.pgdata)"
          '';
        };
      });
    };
}
