import re

file_path = "src/App.vue"
with open(file_path, "r", encoding="utf-8") as f:
    content = f.read()

# 1. Update text to icons for interactions (Like, Boost, Reply, Bookmark)
content = content.replace("回复 {{ post.stats.replies }}", "<MessageCircle class=\"w-[18px] h-[18px] mr-1.5\" /> {{ post.stats.replies || '' }}")
content = content.replace("转发 {{ post.stats.boosts + (boostedPosts[post.id] ? 1 : 0) }}", "<Repeat class=\"w-[18px] h-[18px] mr-1.5\" /> {{ post.stats.boosts + (boostedPosts[post.id] ? 1 : 0) || '' }}")
content = content.replace("喜欢 {{ post.stats.likes + (likedPosts[post.id] ? 1 : 0) }}", "<Heart :class=\"{'fill-current': likedPosts[post.id]}\" class=\"w-[18px] h-[18px] mr-1.5\" /> {{ post.stats.likes + (likedPosts[post.id] ? 1 : 0) || '' }}")
content = content.replace("{{ bookmarkedPosts[post.id] ? '已收藏' : '收藏' }}", "<Bookmark :class=\"{'fill-current': bookmarkedPosts[post.id]}\" class=\"w-[18px] h-[18px]\" />")

content = content.replace("回复 {{ threadFocusPost.stats.replies }}", "<MessageCircle class=\"w-[18px] h-[18px] mr-1.5\" /> {{ threadFocusPost.stats.replies || '' }}")
content = content.replace("转发 {{ threadFocusPost.stats.boosts + (boostedPosts[threadFocusPost.id] ? 1 : 0) }}", "<Repeat class=\"w-[18px] h-[18px] mr-1.5\" /> {{ threadFocusPost.stats.boosts + (boostedPosts[threadFocusPost.id] ? 1 : 0) || '' }}")
content = content.replace("喜欢 {{ threadFocusPost.stats.likes + (likedPosts[threadFocusPost.id] ? 1 : 0) }}", "<Heart :class=\"{'fill-current': likedPosts[threadFocusPost.id]}\" class=\"w-[18px] h-[18px] mr-1.5\" /> {{ threadFocusPost.stats.likes + (likedPosts[threadFocusPost.id] ? 1 : 0) || '' }}")
content = content.replace("{{ bookmarkedPosts[threadFocusPost.id] ? '已收藏' : '收藏' }}", "<Bookmark :class=\"{'fill-current': bookmarkedPosts[threadFocusPost.id]}\" class=\"w-[18px] h-[18px]\" />")

content = content.replace("喜欢 {{ reply.stats.likes + (likedPosts[reply.id] ? 1 : 0) }}", "<Heart :class=\"{'fill-current': likedPosts[reply.id]}\" class=\"w-[18px] h-[18px] mr-1.5\" /> {{ reply.stats.likes + (likedPosts[reply.id] ? 1 : 0) || '' }}")
content = content.replace("{{ bookmarkedPosts[reply.id] ? '已收藏' : '收藏' }}", "<Bookmark :class=\"{'fill-current': bookmarkedPosts[reply.id]}\" class=\"w-[18px] h-[18px]\" />")

content = content.replace("回复此层", "<MessageCircle class=\"w-[18px] h-[18px]\" />")
content = content.replace("查看子讨论", "<List class=\"w-[18px] h-[18px]\" />")

# 2. Add flex to match buttons perfectly
content = content.replace("class=\"rounded-full border border-[color:var(--border-color)] px-3 py-2 text-[color:var(--text-secondary)] transition hover:bg-[var(--chip-hover)] hover:text-[color:var(--text-primary)]\"", "class=\"inline-flex items-center rounded-[2rem] border border-[color:var(--border-color)] px-3 py-1.5 text-sm font-medium text-[color:var(--text-secondary)] transition-all hover:-translate-y-0.5 hover:shadow-sm hover:bg-[var(--chip-hover)] hover:text-[color:var(--text-primary)]\"")
content = content.replace("class=\"rounded-full border px-3 py-2 transition\"", "class=\"inline-flex items-center rounded-[2rem] border px-3 py-1.5 text-sm font-medium transition-all hover:-translate-y-0.5 hover:shadow-sm\"")

# 3. Sidebar Navigation Icons replacement
content = content.replace("<span class=\"w-6 text-center text-lg\">{{ item.icon }}</span>", "<component :is=\"item.icon\" class=\"w-[22px] h-[22px] stroke-[1.5]\" />")
# Wait, some places might have different classes for sidebar icon
content = content.replace("class=\"flex w-full items-center gap-4 rounded-2xl px-4 py-4 text-left text-[18px] transition\"", "class=\"flex w-full items-center gap-4 rounded-[1.5rem] px-4 py-3.5 text-left text-lg font-medium transition-all hover:translate-x-1 hover:bg-[var(--chip-hover)]\"")

# 4. Settings inner sidebar Navigation Icons
content = content.replace("<span class=\"w-5 text-center\">{{ item.icon }}</span>", "<component :is=\"item.icon\" class=\"w-5 h-5 stroke-[1.5]\" />")
content = content.replace("class=\"flex w-full items-center gap-3 rounded-2xl px-4 py-3 text-left text-base transition\"", "class=\"flex w-full items-center gap-3 rounded-[1.5rem] px-4 py-3 text-left text-base font-medium transition-all hover:translate-x-1 hover:bg-[var(--chip-hover)]\"")

# 5. Top Left composer icons replacement
content = content.replace(">⊞<", " title=\"上传图片或视频\"><ImageIcon class=\"w-6 h-6 hover:text-violet-400 cursor-pointer transition-transform hover:scale-110\" /><")
content = content.replace(" title=\"投票\">▤<", " title=\"投票\"><AlignJustify class=\"w-6 h-6 hover:text-violet-400 cursor-pointer transition-transform hover:scale-110\" /><")
content = content.replace(" title=\"预警标签\">△<", " title=\"预警标签\"><AlertTriangle class=\"w-6 h-6 hover:text-amber-400 cursor-pointer transition-transform hover:scale-110\" /><")
content = content.replace(" title=\"表情\">☺<", " title=\"表情\"><Smile class=\"w-6 h-6 hover:text-yellow-400 cursor-pointer transition-transform hover:scale-110\" /><")
# Quick fix for composer label with class
content = content.replace("<label class=\"cursor-pointer transition hover:text-violet-300\" title=\"上传图片或视频\">\n                    ⊞", "<label class=\"cursor-pointer transition hover:text-violet-300\" title=\"上传图片或视频\">\n                    <ImageIcon class=\"w-[22px] h-[22px] stroke-[1.5] transition-transform hover:scale-110\" />")
content = content.replace("<span title=\"投票\">▤</span>", "<span title=\"投票\" class=\"cursor-pointer\"><AlignJustify class=\"w-[22px] h-[22px] stroke-[1.5] transition-transform hover:scale-110\" /></span>")
content = content.replace("<span title=\"预警标签\">△</span>", "<span title=\"预警标签\" class=\"cursor-pointer\"><AlertTriangle class=\"w-[22px] h-[22px] stroke-[1.5] transition-transform hover:scale-110\" /></span>")
content = content.replace("<span title=\"表情\">☺</span>", "<span title=\"表情\" class=\"cursor-pointer\"><Smile class=\"w-[22px] h-[22px] stroke-[1.5] transition-transform hover:scale-110\" /></span>")

# 6. Breathing room modifications - adding rounded-3xl and spacing
# Articles in timeline 
content = content.replace("class=\"px-6 py-6\"", "class=\"px-8 py-8 transition hover:bg-[var(--panel-soft)]\"")

# 7. Remove verbose texts
content = re.sub(r'<div class="space-y-2 pt-4 text-sm text-\[color:var\(--text-muted\)\]">.*?</div>\s*</div>\s*</aside>', '</div>\n        </aside>', content, flags=re.DOTALL)
content = content.replace("服务似乎离家出走了", "本地离线预览模式")

# Fix missing gap between counts and icon
content = content.replace("class=\"w-[18px] h-[18px]\"", "class=\"w-[18px] h-[18px] mr-1.5\"")

# Fix hover:bg mismatch when item selected in sidebar
content = content.replace("class=\"flex w-full items-center gap-4 rounded-[1.5rem] px-4 py-3.5 text-left text-lg font-medium transition-all hover:translate-x-1 hover:bg-[var(--chip-hover)]\"\n                :class=\"currentSection === item.key ? 'bg-violet-600/12 text-violet-600' : 'text-[color:var(--text-secondary)] hover:bg-[var(--chip-bg)]'\"", "class=\"flex w-full items-center gap-4 rounded-[1.5rem] px-4 py-3.5 text-left text-lg font-medium transition-all hover:translate-x-1\"\n                :class=\"currentSection === item.key ? 'bg-violet-600/15 text-violet-600 shadow-sm' : 'text-[color:var(--text-secondary)] hover:bg-[var(--chip-hover)]'\"")

content = content.replace("class=\"flex w-full items-center gap-3 rounded-[1.5rem] px-4 py-3 text-left text-base font-medium transition-all hover:translate-x-1 hover:bg-[var(--chip-hover)]\"\n                    :class=\"currentSettingsTab === item.id ? 'bg-violet-600/12 text-violet-600' : 'text-[color:var(--text-secondary)] hover:bg-[var(--chip-bg)]'\"", "class=\"flex w-full items-center gap-3 rounded-[1.5rem] px-4 py-3 text-left text-base font-medium transition-all hover:translate-x-1\"\n                    :class=\"currentSettingsTab === item.id ? 'bg-violet-600/15 text-violet-600 shadow-sm' : 'text-[color:var(--text-secondary)] hover:bg-[var(--chip-hover)]'\"")

with open(file_path, "w", encoding="utf-8") as f:
    f.write(content)
print("done")
