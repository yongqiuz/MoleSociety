<script setup lang="ts">
import { computed } from 'vue';

type MentionUser = {
  id: string;
  handle: string;
  displayName: string;
  instance?: string;
};

const props = defineProps<{
  text: string;
  users?: MentionUser[];
}>();

const emit = defineEmits<{
  (e: 'open-profile', userId: string): void;
}>();

type Segment =
  | { type: 'text'; value: string }
  | { type: 'mention'; value: string; user?: MentionUser };

function normalizeHandle(value: string) {
  return String(value || '').replace(/^@/, '').trim().toLowerCase();
}

function findUserByMention(raw: string, users: MentionUser[]) {
  const token = String(raw || '').replace(/^@/, '').trim();
  if (!token) return null;
  const [namePart, instancePart] = token.split('@');
  const nameNorm = normalizeHandle(namePart);
  const instanceNorm = String(instancePart || '').trim().toLowerCase();

  const candidates = users.filter((u) => {
    const handleNorm = normalizeHandle(u.handle);
    const displayNorm = String(u.displayName || '').trim().toLowerCase();
    if (handleNorm !== nameNorm && displayNorm !== nameNorm) return false;
    if (!instanceNorm) return true;
    return String(u.instance || '').trim().toLowerCase() === instanceNorm;
  });
  return candidates[0] || null;
}

const segments = computed<Segment[]>(() => {
  const source = String(props.text || '');
  const users = props.users || [];
  const regex = /@([^\s@#，。！？；：、（）()【】\[\]{}<>]{1,32}(?:@[^\s@#，。！？；：、（）()【】\[\]{}<>]{1,32})?)/g;
  const result: Segment[] = [];
  let lastIndex = 0;
  let match: RegExpExecArray | null;
  while ((match = regex.exec(source)) !== null) {
    const start = match.index;
    const full = `@${match[1]}`;
    if (start > lastIndex) {
      result.push({ type: 'text', value: source.slice(lastIndex, start) });
    }
    const user = findUserByMention(full, users) || undefined;
    result.push({ type: 'mention', value: full, user });
    lastIndex = start + full.length;
  }
  if (lastIndex < source.length) {
    result.push({ type: 'text', value: source.slice(lastIndex) });
  }
  return result.length ? result : [{ type: 'text', value: source }];
});
</script>

<template>
  <span class="whitespace-pre-wrap">
    <template v-for="(segment, idx) in segments" :key="idx">
      <template v-if="segment.type === 'text'">{{ segment.value }}</template>
      <button
        v-else-if="segment.user"
        type="button"
        class="inline rounded px-0.5 font-medium text-emerald-500 transition hover:bg-emerald-500/10 hover:text-emerald-400"
        @click.stop="emit('open-profile', segment.user.id)"
      >
        @{{ segment.user.displayName || segment.user.handle.replace(/^@/, '') }}
      </button>
      <template v-else>{{ segment.value }}</template>
    </template>
  </span>
</template>
