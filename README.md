# teambynumbers

Create dummy database:

```bash
mkdir db
cp example.csv db/reports.csv
```

Compile (requires go1.15 or newer) and create a docker image:

```bash
./build.sh
```

Run with:

```bash
docker-compose up
```

You should now be able to visit http://localhost:8888
