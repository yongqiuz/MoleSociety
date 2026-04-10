import os

files_to_update = [
    "src/App.vue"
]

def replace_in_file(filepath):
    if not os.path.exists(filepath):
        return
    with open(filepath, "r", encoding="utf-8") as f:
        content = f.read()

    # Replacing variations of violet with emerald and teal
    content = content.replace("violet-600", "emerald-600")
    content = content.replace("violet-500", "emerald-500")
    content = content.replace("violet-400", "emerald-400")
    content = content.replace("violet-300", "emerald-300")
    content = content.replace("violet-200", "emerald-200")
    
    # Replacing the gradient from-amber-200 to-violet-200
    content = content.replace("from-amber-200 to-violet-200", "from-teal-200 to-emerald-200")
    content = content.replace("from-violet-300 to-cyan-200", "from-emerald-300 to-teal-200")

    with open(filepath, "w", encoding="utf-8") as f:
        f.write(content)

for filepath in files_to_update:
    replace_in_file(filepath)

print("Color replacement successful.")
