<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { useRouter } from 'vue-router';
import { 
  ChevronLeft, 
  Camera, 
  Pencil, 
  Plus, 
  X,
  Info
} from 'lucide-vue-next';
import { useAuth } from '../composables/useAuth';
import { useAppearance } from '../composables/useAppearance';
import { updateUserProfile, type UserField } from '../api/socialApi';

const router = useRouter();
const { currentUser, updateCurrentUserLocally } = useAuth();
const { themeStyles } = useAppearance();

const saving = ref(false);
const error = ref('');

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

async function handleSave() {
  if (!currentUser.value || saving.value) return;
  
  saving.value = true;
  error.value = '';
  
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
    router.push('/app');
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

function goBack() {
  router.back();
}

const avatarText = (name: string) => name.charAt(0).toUpperCase();

</script>

<template>
  <div class="min-h-screen bg-[var(--app-bg)] text-[color:var(--text-primary)] transition-colors duration-300 pb-20" :style="themeStyles">
    <!-- Sticky Header -->
    <header class="sticky top-0 z-10 flex items-center justify-between border-b border-[color:var(--border-color)] bg-[var(--panel-bg)]/80 px-4 py-3 backdrop-blur-md">
      <div class="flex items-center gap-4">
        <button @click="goBack" class="rounded-full p-2 hover:bg-[var(--chip-hover)]">
          <ChevronLeft class="w-6 h-6" />
        </button>
        <h1 class="text-xl font-bold">修改个人资料</h1>
      </div>
      <button 
        @click="handleSave"
        :disabled="saving"
        class="rounded-lg bg-emerald-600 px-6 py-2 text-sm font-bold text-white transition hover:bg-emerald-500 disabled:opacity-50"
      >
        {{ saving ? '保存中...' : '完成' }}
      </button>
    </header>

    <div class="mx-auto max-w-2xl">
      <!-- Avatar Section -->
      <section class="border-b border-[color:var(--border-color)] px-6 py-8">
        <div class="relative inline-block">
          <div class="flex h-24 w-24 items-center justify-center rounded-2xl bg-gradient-to-br from-lime-200 to-cyan-200 text-3xl font-bold text-slate-900 shadow-lg overflow-hidden">
            <template v-if="avatarUrl">
              <img :src="avatarUrl" class="h-full w-full object-cover" />
            </template>
            <template v-else>
              {{ avatarText(displayName || 'U') }}
            </template>
          </div>
          <button class="absolute -bottom-2 -right-2 flex h-8 w-8 items-center justify-center rounded-full border-2 border-[var(--panel-bg)] bg-emerald-600 text-white shadow-md hover:bg-emerald-500 transition">
            <Camera class="w-4 h-4" />
          </button>
        </div>
      </section>

      <!-- Display Name Section -->
      <section class="border-b border-[color:var(--border-color)] px-6 py-6">
        <div class="flex items-center justify-between mb-2">
          <label class="text-sm font-bold text-[color:var(--text-primary)]">显示名称</label>
          <Pencil class="w-4 h-4 text-[color:var(--text-muted)]" />
        </div>
        <input 
          v-model="displayName"
          type="text"
          placeholder="你的称呼"
          class="w-full bg-transparent text-lg font-medium text-[color:var(--text-primary)] outline-none border-b border-transparent focus:border-emerald-500 transition-colors py-1"
        />
      </section>

      <!-- Bio Section -->
      <section class="border-b border-[color:var(--border-color)] px-6 py-6">
        <div class="flex items-center justify-between mb-4">
          <label class="text-sm font-bold text-[color:var(--text-primary)]">简介</label>
          <button @click="bio = ''" v-if="bio" class="text-xs text-emerald-500 font-medium">重置</button>
          <button v-else class="rounded-lg border border-[color:var(--border-color)] px-3 py-1.5 text-xs font-medium hover:bg-[var(--chip-hover)]">添加个人简介</button>
        </div>
        <textarea 
          v-model="bio"
          placeholder="添加一段简短介绍，帮助其他人认识你。"
          class="w-full min-h-[100px] bg-transparent text-base text-[color:var(--text-secondary)] outline-none resize-none leading-relaxed"
        ></textarea>
      </section>

      <!-- Custom Fields Section -->
      <section class="border-b border-[color:var(--border-color)] px-6 py-6">
        <div class="flex items-center justify-between mb-4">
          <label class="text-sm font-bold text-[color:var(--text-primary)]">自定义字段</label>
          <button 
            @click="addField"
            class="rounded-lg border border-[color:var(--border-color)] px-3 py-1.5 text-xs font-medium hover:bg-[var(--chip-hover)] flex items-center gap-1"
          >
            <Plus class="w-3 h-3" />
            添加字段
          </button>
        </div>
        <p class="text-xs text-[color:var(--text-muted)] mb-4">添加你的人身代词、外部链接，或其他你想分享的内容。</p>
        
        <div class="space-y-3">
          <div v-for="(field, index) in fields" :key="index" class="flex gap-2">
            <input 
              v-model="field.name" 
              placeholder="标签（如：网站）" 
              class="w-1/3 rounded-xl border border-[color:var(--border-color)] bg-[var(--panel-soft)] px-3 py-2 text-sm outline-none focus:border-emerald-500"
            />
            <input 
              v-model="field.value" 
              placeholder="内容（如：https://...）" 
              class="flex-1 rounded-xl border border-[color:var(--border-color)] bg-[var(--panel-soft)] px-3 py-2 text-sm outline-none focus:border-emerald-500"
            />
            <button @click="removeField(index)" class="p-2 text-rose-500 hover:bg-rose-500/10 rounded-lg">
              <X class="w-4 h-4" />
            </button>
          </div>
        </div>

        <!-- Verification Hint -->
        <div v-if="showHint" class="mt-4 rounded-2xl bg-emerald-500/5 border border-emerald-500/10 p-4 relative">
          <button @click="showHint = false" class="absolute top-3 right-3 text-[color:var(--text-muted)]">
            <X class="w-4 h-4" />
          </button>
          <div class="flex gap-3">
            <div class="mt-0.5 rounded-full bg-emerald-500/10 p-1.5">
              <Info class="w-4 h-4 text-emerald-500" />
            </div>
            <div class="pr-6">
              <div class="text-sm font-bold text-[color:var(--text-primary)] mb-1">小贴士：添加已验证的链接</div>
              <p class="text-xs text-[color:var(--text-secondary)] leading-relaxed">
                通过验证任意你拥有网站的链接，你可以轻松增加 Mastodon 账号的可信度。
              </p>
            </div>
          </div>
        </div>
      </section>

      <!-- Featured Tags Section -->
      <section class="border-b border-[color:var(--border-color)] px-6 py-6">
        <div class="flex items-center justify-between mb-4">
          <label class="text-sm font-bold text-[color:var(--text-primary)]">精选话题标签</label>
          <button 
            @click="addTag"
            class="rounded-lg border border-[color:var(--border-color)] px-3 py-1.5 text-xs font-medium hover:bg-[var(--chip-hover)] flex items-center gap-1"
          >
            <Plus class="w-3 h-3" />
            添加话题标签
          </button>
        </div>
        <p class="text-xs text-[color:var(--text-muted)] mb-4">帮助其他人认识并快速访问你最喜欢的话题。</p>
        
        <div class="flex flex-wrap gap-2">
          <div v-for="tag in featuredTags" :key="tag" class="flex items-center gap-1 rounded-full bg-emerald-500/10 border border-emerald-500/20 px-3 py-1 text-sm text-emerald-600">
            <span>#{{ tag }}</span>
            <button @click="removeTag(tag)" class="hover:text-rose-500"><X class="w-3 h-3" /></button>
          </div>
        </div>
      </section>

      <!-- Profile Tabs Section -->
      <section class="border-b border-[color:var(--border-color)] px-6 py-6">
        <div class="flex items-center justify-between mb-4">
          <label class="text-sm font-bold text-[color:var(--text-primary)]">个人资料标签页设置</label>
          <button class="rounded-lg border border-[color:var(--border-color)] px-3 py-1.5 text-xs font-medium hover:bg-[var(--chip-hover)]">自定义</button>
        </div>
        <p class="text-xs text-[color:var(--text-muted)]">自定义你个人资料的标签页及其显示的内容。</p>
      </section>

      <!-- Advanced Settings Section -->
      <section class="px-6 py-8">
        <h2 class="text-lg font-bold mb-6">高级设置</h2>
        <div class="flex items-center justify-between">
          <div>
            <div class="text-sm font-bold text-[color:var(--text-primary)] mb-1">机器人账号</div>
            <p class="text-xs text-[color:var(--text-muted)]">来自这个账号的绝大多数操作都是自动进行的，并且可能无人监控。</p>
          </div>
          <button 
            @click="isBot = !isBot"
            class="relative h-6 w-11 rounded-full transition-colors duration-200 focus:outline-none"
            :class="isBot ? 'bg-emerald-600' : 'bg-slate-300 dark:bg-slate-700'"
          >
            <span 
              class="absolute left-1 top-1 h-4 w-4 rounded-full bg-white transition-transform duration-200"
              :class="isBot ? 'translate-x-5' : 'translate-x-0'"
            ></span>
          </button>
        </div>
      </section>
    </div>
  </div>
</template>
