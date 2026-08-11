import os
import re

def find_swallowed_errors(directory):
    pattern = re.compile(r'if\s+err\s*!=\s*nil\s*{\s*}', re.MULTILINE)
    for root, _, files in os.walk(directory):
        for file in files:
            if file.endswith('.go'):
                filepath = os.path.join(root, file)
                with open(filepath, 'r', encoding='utf-8') as f:
                    content = f.read()
                    if pattern.search(content):
                        print(f"Empty err check found in {filepath}")

if __name__ == '__main__':
    find_swallowed_errors(r'c:\Users\user\Desktop\lfx2')
