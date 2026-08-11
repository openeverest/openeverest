import urllib.request
import json
import sys

def search_github(repo, query):
    url = f'https://api.github.com/search/issues?q=repo:{repo}+{urllib.parse.quote(query)}'
    req = urllib.request.Request(url, headers={'User-Agent': 'Mozilla/5.0'})
    try:
        with urllib.request.urlopen(req) as response:
            data = json.loads(response.read().decode())
            print(f'{repo} matches for "{query}": {data["total_count"]}')
            for item in data.get('items', []):
                print(f"- {item['number']}: {item['title']} ({item['state']})")
    except Exception as e:
        print(f"Error querying {repo}: {e}")

search_github('percona/everest', 'CreateBackupStorage Secret')
search_github('percona/everest', 'DeleteBackupStorage Secret')
search_github('percona/everest', 'CreateSecret BackupStorage')
search_github('percona/everest', 'backup storage secret overwritten')
search_github('percona/everest', 'backup storage secret deleted')
search_github('openeverest/openeverest', 'CreateBackupStorage Secret')
search_github('openeverest/openeverest', 'backup storage secret deleted')
