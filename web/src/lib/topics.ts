// Cambridge A2 Key / B1 Preliminary topic slugs (must match the API) with Vietnamese labels.
export const TOPICS: { id: string; label: string }[] = [
  { id: 'clothes', label: 'Quần áo' },
  { id: 'communication-and-technology', label: 'Giao tiếp & công nghệ' },
  { id: 'education', label: 'Giáo dục' },
  { id: 'entertainment-and-media', label: 'Giải trí & truyền thông' },
  { id: 'environment', label: 'Môi trường' },
  { id: 'food-and-drink', label: 'Ăn uống' },
  { id: 'health', label: 'Sức khỏe' },
  { id: 'hobbies-and-leisure', label: 'Sở thích' },
  { id: 'house-and-home', label: 'Nhà cửa' },
  { id: 'language', label: 'Ngôn ngữ' },
  { id: 'personal-feelings', label: 'Cảm xúc' },
  { id: 'places', label: 'Địa điểm' },
  { id: 'services', label: 'Dịch vụ' },
  { id: 'shopping', label: 'Mua sắm' },
  { id: 'social-interaction', label: 'Giao tiếp xã hội' },
  { id: 'sport', label: 'Thể thao' },
  { id: 'the-natural-world', label: 'Thiên nhiên' },
  { id: 'transport', label: 'Giao thông' },
  { id: 'travel-and-holidays', label: 'Du lịch' },
  { id: 'weather', label: 'Thời tiết' },
  { id: 'work-and-jobs', label: 'Công việc' },
  { id: 'general', label: 'Chung' },
]

export const topicLabel = (id: string) => TOPICS.find((t) => t.id === id)?.label ?? id
