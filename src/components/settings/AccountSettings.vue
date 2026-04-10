<script setup lang="ts">
import { ref, onMounted, computed } from 'vue';
import { 
  Camera, 
  Pencil, 
  Plus, 
  X,
  Info
} from 'lucide-vue-next';
import { useAuth } from '../../composables/useAuth';
import { updateUserProfile, type UserField } from '../../api/socialApi';

const { currentUser, updateCurrentUserLocally } = useAuth();

const saving = ref(false);
const error = ref('');
const notice = ref('');

// Form state
const displayName = ref('');
const bio = ref('');
const avatarUrl = ref('');
const fields = ref<UserField[]>([]);
const featuredTags = ref<string[]>([]);
const isBot = ref(false);

const showHint = ref(true);

onMounted(() => {
  if (currentUser.value) {
    displayName.value = currentUser.value.displayName || '';
    bio.value = currentUser.value.bio || '';
    avatarUrl.value = currentUser.value.avatarUrl || '';
    fields.value = currentUser.value.fields ? [...currentUser.value.fields] : [];
    featuredTags.value = currentUser.value.featuredTags ? [...currentUser.value.featuredTags] : [];
    isBot.value = currentUser.value.isBot || false;
  }
});

const hasChanges = computed(() => {
  if (!currentUser.value) return false;
  return displayName.value !== (currentUser.value.displayName || '') ||
         bio.value !== (currentUser.value.bio || '') ||
         avatarUrl.value !== (currentUser.value.avatarUrl || '') ||
         JSON.stringify(fields.value) !== JSON.stringify(currentUser.value.fields || []) ||
         JSON.stringify(featuredTags.value) !== JSON.stringify(currentUser.value.featuredTags || []) ||
         isBot.value !== (currentUser.value.isBot || false);
});

async function handleSave() {
  if (!currentUser.value || saving.value) return;
  
  saving.value = true;
  error.value = '';
  notice.value = '';
  
  try {
    const updatedUser = await updateUserProfile(currentUser.value.id, {
      displayName: displayName.value,
      bio: bio.value,
      avatarUrl: avatarUrl.value,
      fields: fields.value,
      featuredTags: featuredTags.value,
      isBot: isBot.value,
    });
    
    updateCurrentUserLocally(updatedUser);
    notice.value = '个人资料已更新';
    setTimeout(() => notice.value = '', 3000);
  } catch (err: any) {
    error.value = err.message || '保存失败';
  } finally {
    saving.value = false;
  }
}

function addField() {
  fields.value.push({ name: '', value: '' });
}

function removeField(index: number) {
  fields.value.splice(index, 1);
}

function addTag() {
  const tag = prompt('输入话题标签（无需 # 号）');
  if (tag && !featuredTags.value.includes(tag)) {
    featuredTags.value.push(tag);
  }
}

function removeTag(tag: string) {
  featuredTags.value = featuredTags.value.filter(t => t !== tag);
}

const avatarText = (name: string) => name ? name.charAt(0).toUpperCase() : 'U';
</script>

<template>
  <div class="px-8 py-8 lg:px-10 max-w-3xl">
    <div class="mb-10 flex flex-wrap items-center justify-between gap-4">
      <div>
        <div class="text-[34px] font-semibold tracking-tight text-[color:var(--text-primary)]">个人资料</div>
        <div class="mt-2 text-sm text-[color:var(--text-muted)]">管理你的公开个人信息、头像、简介及精选话题。</div>
      </div>

      <div class="flex items-center gap-3">
        <span v-if="notice" class="text-sm text-emerald-500 font-medium">{{ notice }}</span>
        <span v-if="error" class="text-sm text-rose-500 font-medium">{{ error }}</span>
        <button 
          @click="handleSave"
          :disabled="saving || !hasChanges"
          class="rounded-xl bg-emerald-600 px-6 py-3 text-base font-semibold text-white transition hover:bg-emerald-500 disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {{ saving ? '保存中...' : '保存更改' }}
        </button>
      </div>
    </div>

    <!-- Avatar Section -->
    <section class="mb-10 rounded-3xl border border-[color:var(--border-color)] bg-[var(--panel-soft)] p-8">
      <div class="flex items-center gap-8">
        <div class="relative">
          <div class="flex h-24 w-24 items-center justify-center rounded-2xl bg-gradient-to-br from-lime-200 to-cyan-200 text-3xl font-bold text-slate-900 shadow-sm overflow-hidden">
            <template v-if="avatarUrl">
              <img :src="avatarUrl" class="h-full w-full object-cover" />
            </template>
            <template v-else>
              {{ avatarText(displayName) }}
            </template>
          </div>
          <button class="absolute -bottom-2 -right-2 flex h-8 w-8 items-center justify-center rounded-full border-2 border-[var(--panel-soft)] bg-emerald-600 text-white shadow-md hover:bg-emerald-500 transition">
            <Camera class="w-4 h-4" />
          </button>
        </div>
        <div>
          <div class="text-lg font-bold text-[color:var(--text-primary)]">头像</div>
          <p class="mt-1 text-sm text-[color:var(--text-muted)]">建议使用正方形图片 400x400px。</p>
        </div>
      </div>
    </section>

    <!-- Form Sections -->
    <div class="space-y-8 pb-10">
      <!-- Display Name -->
      <div class="rounded-3xl border border-[color:var(--border-color)] bg-[var(--panel-soft)] p-8">
        <div class="flex items-center justify-between mb-4">
          <label class="text-sm font-bold text-[color:var(--text-primary)] uppercase tracking-wider">显示名称</label>
          <Pencil class="w-4 h-4 text-[color:var(--text-muted)]" />
        </div>
        <input 
          v-model="displayName"
          type="text"
          class="w-full bg-transparent text-xl font-medium text-[color:var(--text-primary)] outline-none border-b border-[color:var(--border-color)] focus:border-emerald-500 transition-colors py-2"
          placeholder="你的称呼"
        />
      </div>

      <!-- Bio -->
      <div class="rounded-3xl border border-[color:var(--border-color)] bg-[var(--panel-soft)] p-8">
        <div class="flex items-center justify-between mb-4">
          <label class="text-sm font-bold text-[color:var(--text-primary)] uppercase tracking-wider">个人简介</label>
          <button @click="bio = ''" v-if="bio" class="text-xs text-emerald-500 font-medium hover:underline">清空</button>
        </div>
        <textarea 
          v-model="bio"
          placeholder="添加一段简短介绍，帮助其他人认识你。"
          class="w-full min-h-[120px] bg-transparent text-base text-[color:var(--text-secondary)] outline-none resize-none leading-relaxed border-b border-[color:var(--border-color)] focus:border-emerald-500 transition-colors py-2"
        ></textarea>
      </div>

      <!-- Custom Fields -->
      <div class="rounded-3xl border border-[color:var(--border-color)] bg-[var(--panel-soft)] p-8">
        <div class="flex items-center justify-between mb-6">
          <label class="text-sm font-bold text-[color:var(--text-primary)] uppercase tracking-wider">自定义字段</label>
          <button 
            @click="addField"
            class="rounded-xl border border-[color:var(--border-color)] bg-[var(--panel-muted)] px-4 py-2 text-sm font-medium hover:bg-[var(--chip-hover)] transition-colors inline-flex items-center gap-2 text-[color:var(--text-primary)]"
          >
            <Plus class="w-4 h-4" />
            添加字段
          </button>
        </div>
        
        <div class="space-y-4">
          <div v-for="(field, index) in fields" :key="index" class="flex items-center gap-3">
            <input 
              v-model="field.name" 
              placeholder="标签 (如：网站)" 
              class="w-1/3 rounded-xl border border-[color:var(--border-color)] bg-[var(--panel-bg)] px-4 py-2.5 text-sm outline-none focus:border-emerald-500 transition-colors font-medium text-[color:var(--text-primary)]"
            />
            <input 
              v-model="field.value" 
              placeholder="内容" 
              class="flex-1 rounded-xl border border-[color:var(--border-color)] bg-[var(--panel-bg)] px-4 py-2.5 text-sm outline-none focus:border-emerald-500 transition-colors text-[color:var(--text-primary)]"
            />
            <button @click="removeField(index)" class="p-2.5 text-rose-500 hover:bg-rose-500/10 rounded-xl transition-colors">
              <X class="w-5 h-5" />
            </button>
          </div>
        </div>

        <div v-if="showHint" class="mt-8 rounded-2xl bg-emerald-500/5 border border-emerald-500/15 p-5 relative overflow-hidden group">
          <button @click="showHint = false" class="absolute top-4 right-4 text-[color:var(--text-muted)] hover:text-emerald-500 transition-colors">
            <X class="w-4 h-4" />
          </button>
          <div class="flex gap-4">
            <div class="shrink-0 mt-1 rounded-full bg-emerald-500/10 p-2">
              <Info class="w-4 h-4 text-emerald-500" />
            </div>
            <div>
              <div class="text-sm font-bold text-[color:var(--text-primary)] mb-1.5">小贴士：添加已验证的链接</div>
              <p class="text-xs text-[color:var(--text-secondary)] leading-relaxed">
                通过验证任何你拥有所有权的网站链接，你可以显著增加账号的可信度。
              </p>
            </div>
          </div>
        </div>
      </div>

      <!-- Featured Tags -->
      <div class="rounded-3xl border border-[color:var(--border-color)] bg-[var(--panel-soft)] p-8">
        <div class="flex items-center justify-between mb-6">
          <label class="text-sm font-bold text-[color:var(--text-primary)] uppercase tracking-wider">精选话题标签</label>
          <button 
            @click="addTag"
            class="rounded-xl border border-[color:var(--border-color)] bg-[var(--panel-muted)] px-4 py-2 text-sm font-medium hover:bg-[var(--chip-hover)] transition-colors inline-flex items-center gap-2 text-[color:var(--text-primary)]"
          >
            <Plus class="w-4 h-4" />
            添加话题
          </button>
        </div>
        
        <div class="flex flex-wrap gap-2.5">
          <div v-for="tag in featuredTags" :key="tag" class="flex items-center gap-2 rounded-full bg-emerald-500/10 border border-emerald-500/20 px-4 py-1.5 text-sm font-medium text-emerald-600">
            <span>#{{ tag }}</span>
            <button @click="removeTag(tag)" class="hover:text-rose-500 transition-colors">
              <X class="w-3.5 h-3.5" />
            </button>
          </div>
          <div v-if="featuredTags.length === 0" class="text-sm text-[color:var(--text-muted)] italic">
            尚未添加精选话题。
          </div>
        </div>
      </div>

      <!-- Robot Account switch -->
      <div class="rounded-3xl border border-[color:var(--border-color)] bg-[var(--panel-soft)] p-8">
         <div class="flex items-center justify-between">
          <div class="pr-8">
            <div class="text-base font-bold text-[color:var(--text-primary)] mb-1.5">机器人账号</div>
            <p class="text-sm text-[color:var(--text-muted)] leading-relaxed">标记为一个由自动化脚本/算法驱动的账号，帮助用户识别非人工互动。</p>
          </div>
          <button 
            @click="isBot = !isBot"
            class="relative h-7 w-12 rounded-full transition-all duration-300 focus:outline-none"
            :class="isBot ? 'bg-emerald-600 shadow-[0_0_12px_rgba(5,150,105,0.4)]' : 'bg-slate-300 dark:bg-slate-700'"
          >
            <span 
              class="absolute left-1 top-1 h-5 w-5 rounded-full bg-white transition-transform duration-300 shadow-sm"
              :class="isBot ? 'translate-x-5' : 'translate-x-0'"
            ></span>
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
