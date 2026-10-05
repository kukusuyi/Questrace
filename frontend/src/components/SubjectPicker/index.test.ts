import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { reactive,defineComponent } from 'vue'
import { beforeEach, expect, it,vi } from 'vitest'
import SubjectPicker from './index.vue'
import {useSubjectsStore,type Classification} from '@/stores/subjects.store'
vi.mock('@/api/http',()=>({httpGet:vi.fn(async()=>({list:[{id:'cs408',name:'408',specialized:true,chapters:[],courses:[{id:'data_structures',name:'数据结构',chapters:['排序']},{id:'operating_systems',name:'操作系统',chapters:['内存管理']}]},{id:'highschool_physics',name:'高中物理',specialized:false,chapters:[],courses:[]}]})),httpPost:vi.fn(async()=>({id:'custom_one',name:'天文学',courses:[],chapters:[],specialized:false}))}))
const Select=defineComponent({name:'ElSelect',props:['modelValue'],emits:['change'],template:'<div><slot/></div>'})
beforeEach(()=>setActivePinia(createPinia()))
it('clears incompatible chapter/course and analysis confirmation when subject changes',async()=>{
 const model=reactive<Classification>({subject_id:'cs408',course_id:'data_structures',chapter:'排序',analysis_confirmed:true});const wrapper=mount(SubjectPicker,{props:{model},global:{stubs:{ElButton:true,ElInput:true,ElSelect:Select,ElOption:true,ElFormItem:{template:'<div><slot/></div>'}}}});await flushPromises();
 wrapper.findAllComponents(Select)[0]!.vm.$emit('change','highschool_physics');await flushPromises();expect(model.subject_id).toBe('highschool_physics');expect(model.course_id).toBe('');expect(model.chapter).toBe('');expect(model.analysis_confirmed).toBe(false);expect(model.analysis_stale).toBe(true);
})
it('uses only chapters belonging to the current 408 course',async()=>{
 const model=reactive<Classification>({subject_id:'cs408',course_id:'data_structures',chapter:'排序'});const wrapper=mount(SubjectPicker,{props:{model},global:{stubs:{ElButton:true,ElInput:true,ElSelect:Select,ElOption:{props:['label'],template:'<span>{{label}}</span>'},ElFormItem:{template:'<div><slot/></div>'}}}});await flushPromises();expect(wrapper.text()).toContain('排序');expect(wrapper.text()).not.toContain('内存管理');wrapper.findAllComponents(Select)[1]!.vm.$emit('change','operating_systems');await flushPromises();expect(model.chapter).toBe('');expect(wrapper.text()).toContain('内存管理');expect(wrapper.text()).not.toContain('排序');
})

it('recommends only the current stage while preserving existing classifications',async()=>{
 const model=reactive<Classification>({subject_id:''});
 const wrapper=mount(SubjectPicker,{props:{model},global:{stubs:{ElButton:true,ElInput:true,ElSelect:Select,ElOption:{props:['label'],template:'<span>{{label}}</span>'},ElFormItem:{template:'<div><slot/></div>'}}}});
 await flushPromises();const store=useSubjectsStore();
 store.items=[{id:'cs408',name:'408',education_stage:'university',courses:[],chapters:[],specialized:true},{id:'highschool_physics',name:'高中物理',education_stage:'highschool',courses:[],chapters:[],specialized:false},{id:'custom_materials',name:'材料科学基础',education_stage:'university',courses:[],chapters:[],specialized:false}];
 store.educationStage='university';await flushPromises();expect(wrapper.text()).toContain('材料科学基础');expect(wrapper.text()).not.toContain('高中物理');
 store.educationStage='highschool';await flushPromises();expect(wrapper.text()).toContain('高中物理');expect(wrapper.text()).not.toContain('408');expect(wrapper.text()).not.toContain('材料科学基础');
 model.subject_id='custom_materials';await flushPromises();expect(wrapper.text()).toContain('材料科学基础');expect(model.subject_id).toBe('custom_materials');
})

it('prefills an AI suggestion and creates it only after explicit confirmation',async()=>{
 const {httpPost}=await import('@/api/http');vi.mocked(httpPost).mockClear();
 const model=reactive<Classification>({subject_id:'',suggested_subject:'材料力学',classification_status:'pending'});
 const Input=defineComponent({name:'ElInput',props:['modelValue'],emits:['update:modelValue'],template:'<input :value="modelValue" @input="$emit(\'update:modelValue\', $event.target.value)"/>'});
 const Button=defineComponent({name:'ElButton',props:['disabled'],emits:['click'],template:'<button :disabled="disabled" @click="$emit(\'click\')"><slot/></button>'});
 const wrapper=mount(SubjectPicker,{props:{model},global:{stubs:{ElInput:Input,ElButton:Button,ElSelect:Select,ElOption:true,ElFormItem:{template:'<div><slot/></div>'}}}});
 await flushPromises();expect(wrapper.find('input').element.value).toBe('材料力学');expect(httpPost).not.toHaveBeenCalled();expect(model.subject_id).toBe('');
 await wrapper.find('input').setValue('机械原理');expect(model.suggested_subject).toBe('机械原理');
 vi.mocked(httpPost).mockRejectedValueOnce(new Error('network unavailable'));
 await wrapper.findAll('button').find(b=>b.text()==='确认学科')!.trigger('click');await flushPromises();
 expect(wrapper.find('input').element.value).toBe('机械原理');expect(model.subject_id).toBe('');
 vi.mocked(httpPost).mockResolvedValueOnce({id:'custom_mechanical',name:'机械原理',courses:[],chapters:[],specialized:false});
 await wrapper.findAll('button').find(b=>b.text()==='确认学科')!.trigger('click');await flushPromises();
 expect(httpPost).toHaveBeenLastCalledWith('/api/v1/subjects',{name:'机械原理'});expect(model.subject_id).toBe('custom_mechanical');expect(model.suggested_subject).toBe('');expect(model.classification_status).toBe('confirmed');expect(useSubjectsStore().items.some(s=>s.id==='custom_mechanical')).toBe(true);
})
