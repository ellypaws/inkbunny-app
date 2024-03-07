// import LoginForm from '@/LoginForm';
import Page from '@/app/page';
import {BackgroundBeams} from "@/components/background-beams.tsx";

export function App() {
    return (
        <div className="lg:pb-24">
            {/*<LoginForm/>*/}
            <Page/>
            <BackgroundBeams className="fixed inset-0 z-0 pointer-events-none" />
        </div>
    );
}

export default App;
